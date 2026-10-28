package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type processResource struct{}

// The framework constructs a fresh resource instance for each RPC. Keep started
// commands in the provider process and identify them through ephemeral private data.
var processes sync.Map

type processModel struct {
	Command               types.List   `tfsdk:"command"`
	WorkingDir            types.String `tfsdk:"working_directory"`
	Environment           types.Map    `tfsdk:"environment"`
	WaitForStartupDelayMs types.Int64  `tfsdk:"wait_for_startup_delay_ms"`
	WaitForOutputRegex    types.String `tfsdk:"wait_for_output_regex"`
	WaitTimeoutMs         types.Int64  `tfsdk:"wait_timeout_ms"`
	WaitForListeningPort  types.List   `tfsdk:"wait_for_listening_port"`
	Pid                   types.Int64  `tfsdk:"pid"`
}

type portModel struct {
	Port     types.Int64  `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
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
			"working_directory":         schema.StringAttribute{Description: "Working directory for the process.", Optional: true},
			"environment":               schema.MapAttribute{Description: "Environment variables added to the current environment.", ElementType: types.StringType, Optional: true},
			"wait_for_startup_delay_ms": schema.Int64Attribute{Description: "Wait this many non-negative milliseconds after starting.", Optional: true},
			"wait_for_output_regex":     schema.StringAttribute{Description: "Wait for a complete stdout or stderr line matching this regex.", Optional: true},
			"wait_timeout_ms":           schema.Int64Attribute{Description: "Optional timeout for the entire wait after process start", Optional: true},
			"pid":                       schema.Int64Attribute{Description: "Operating system process ID.", Computed: true},
		},
		Blocks: map[string]schema.Block{
			"wait_for_listening_port": schema.ListNestedBlock{
				Description: "Wait for a TCP connection to each configured port on 127.0.0.1.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"port":     schema.Int64Attribute{Required: true, Description: "Local TCP port, from 1 to 65535."},
					"protocol": schema.StringAttribute{Required: true, Description: "Protocol; currently only tcp is supported."},
				}},
			},
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
	config := waitConfig{}
	config.delayMs = data.WaitForStartupDelayMs.ValueInt64Pointer()
	if config.delayMs != nil && *config.delayMs < 0 {
		resp.Diagnostics.AddError("Invalid startup delay", "wait_for_startup_delay_ms must be non-negative")
		return
	}
	config.timeoutMs = data.WaitTimeoutMs.ValueInt64Pointer()
	if config.timeoutMs != nil && *config.timeoutMs < 0 {
		resp.Diagnostics.AddError("Invalid wait timeout", "wait_timeout_ms must be non-negative")
		return
	}
	waitForOutputRegexVSP := data.WaitForOutputRegex.ValueStringPointer()
	if waitForOutputRegexVSP != nil {
		var err error
		config.outputRegex, err = regexp.Compile(*waitForOutputRegexVSP)
		if err != nil {
			resp.Diagnostics.AddError("Invalid output regex", err.Error())
			return
		}
	}
	if !data.WaitForListeningPort.IsNull() {
		var ports []portModel
		resp.Diagnostics.Append(data.WaitForListeningPort.ElementsAs(ctx, &ports, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, port := range ports {
			if port.Port.ValueInt64() < 1 || port.Port.ValueInt64() > 65535 {
				resp.Diagnostics.AddError("Invalid listening port", "port must be between 1 and 65535")
				return
			}
			if port.Protocol.ValueString() != "tcp" {
				resp.Diagnostics.AddError(
					"Unsupported protocol",
					"wait_for_listening_port supports only tcp; UDP cannot confirm readiness",
				)
				return
			}
			config.ports = append(config.ports, int(port.Port.ValueInt64()))
		}
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
	matched := make(chan struct{}, 1)
	cmd.Stdout = newLineMatcher(config.outputRegex, matched)
	cmd.Stderr = newLineMatcher(config.outputRegex, matched)
	cmd.WaitDelay = time.Second // Bound inherited output pipes after the direct child exits.
	if err := cmd.Start(); err != nil {
		resp.Diagnostics.AddError("Failed to start process", err.Error())
		return
	}
	process := startProcess(cmd, func(err error) {
		tflog.Debug(context.WithoutCancel(ctx), "Subprocess exited before Close", map[string]any{
			"pid":         cmd.Process.Pid,
			"exit_status": processExit(err),
		})
	})
	if err := waitForReady(ctx, process, config, matched); err != nil {
		_ = process.stop()
		resp.Diagnostics.AddError("Process readiness failed", err.Error())
		return
	}
	data.Pid = types.Int64Value(int64(cmd.Process.Pid))
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, "pid", []byte(strconv.Itoa(cmd.Process.Pid)))...)
	}
	if resp.Diagnostics.HasError() {
		_ = process.stop()
		return
	}
	if err := process.markOpened(); err != nil {
		_ = process.stop()
		resp.Diagnostics.AddError("Process exited during Open", err.Error())
		return
	}
	processes.Store(cmd.Process.Pid, process)
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
	process := stored.(*runningProcess)
	if err := process.stop(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		resp.Diagnostics.AddWarning("Failed to stop process", fmt.Sprintf("PID %d: %s", process.cmd.Process.Pid, err))
	}
}
