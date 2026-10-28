package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type processResource struct{}

// The framework constructs a fresh resource instance for each RPC. Keep started
// commands in the provider process and identify them through ephemeral private data.
var processes sync.Map

type processModel struct {
	Command        types.List   `tfsdk:"command"`
	WorkingDir     types.String `tfsdk:"working_directory"`
	Environment    types.Map    `tfsdk:"environment"`
	StartupDelayMs types.Int64  `tfsdk:"startup_delay_ms"`
	Pid            types.Int64  `tfsdk:"pid"`
}

var _ ephemeral.EphemeralResource = (*processResource)(nil)
var _ ephemeral.EphemeralResourceWithClose = (*processResource)(nil)

func (r *processResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_process"
}

func (r *processResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Starts a local process when this ephemeral resource opens and kills it when it closes. Use depends_on to order other operations after it.",
		Attributes: map[string]schema.Attribute{
			"command": schema.ListAttribute{
				Description: "Executable and arguments, without a shell. For shell features, explicitly invoke a shell (for example, [\"sh\", \"-c\", \"...\"]).",
				ElementType: types.StringType,
				Required:    true,
			},
			"working_directory": schema.StringAttribute{Description: "Working directory for the process.", Optional: true},
			"environment":       schema.MapAttribute{Description: "Environment variables added to the current environment.", ElementType: types.StringType, Optional: true},
			"startup_delay_ms":  schema.Int64Attribute{Description: "Wait this many milliseconds after starting before returning; defaults to zero.", Optional: true},
			"pid":               schema.Int64Attribute{Description: "Operating system process ID.", Computed: true},
		},
	}
}

func (r *processResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data processModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var args []string
	resp.Diagnostics.Append(data.Command.ElementsAs(ctx, &args, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(args) == 0 || args[0] == "" {
		resp.Diagnostics.AddError("Invalid command", "command must contain an executable name")
		return
	}

	cmd := exec.Command(args[0], args[1:]...) // #nosec G204 -- executing the configured command is the purpose of this provider
	if !data.WorkingDir.IsNull() {
		cmd.Dir = data.WorkingDir.ValueString()
	}
	if !data.Environment.IsNull() {
		var env map[string]string
		resp.Diagnostics.Append(data.Environment.ElementsAs(ctx, &env, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		cmd.Env = os.Environ()
		keys := make([]string, 0, len(env))
		for key := range env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			cmd.Env = append(cmd.Env, key+"="+env[key])
		}
	}
	// stdout and stderr intentionally inherit /dev/null: provider stdout is the RPC transport.
	if err := cmd.Start(); err != nil {
		resp.Diagnostics.AddError("Failed to start process", err.Error())
		return
	}
	if !data.StartupDelayMs.IsNull() {
		delay := data.StartupDelayMs.ValueInt64()
		if delay < 0 || delay > 300000 {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			resp.Diagnostics.AddError("Invalid startup delay", "startup_delay_ms must be between 0 and 300000")
			return
		}
		select {
		case <-time.After(time.Duration(delay) * time.Millisecond):
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			resp.Diagnostics.AddError("Process start cancelled", ctx.Err().Error())
			return
		}
	}
	data.Pid = types.Int64Value(int64(cmd.Process.Pid))
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, "pid", []byte(strconv.Itoa(cmd.Process.Pid)))...)
	}
	if resp.Diagnostics.HasError() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return
	}
	processes.Store(cmd.Process.Pid, cmd)
}

func (r *processResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	value, diags := req.Private.GetKey(ctx, "pid")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	pid, err := strconv.Atoi(string(value))
	if err != nil {
		return
	}
	stored, ok := processes.LoadAndDelete(pid)
	if !ok {
		return
	}
	cmd := stored.(*exec.Cmd)
	if err := cmd.Process.Kill(); err != nil && err != os.ErrProcessDone {
		resp.Diagnostics.AddWarning("Failed to stop process", fmt.Sprintf("PID %d: %s", cmd.Process.Pid, err))
	}
	_ = cmd.Wait()
}
