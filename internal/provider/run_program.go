package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func runProgram(ctx context.Context, kind string, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	suffix := ""
	if kind != "" {
		suffix = fmt.Sprintf("_%s", kind)
	}

	actionVal := ctx.Value("tf_action")
	action := ""
	if actionVal != nil {
		action = actionVal.(string)
	}

	programAttr := fmt.Sprintf("program%s", suffix)
	workingDirAttr := fmt.Sprintf("working_dir%s", suffix)
	queryAttr := fmt.Sprintf("query%s", suffix)

	programI := d.Get(programAttr).([]interface{})
	if programI == nil || len(programI) == 0 {
		return nil
	}

	workingDir := d.Get(workingDirAttr).(string)
	query := d.Get(queryAttr).(map[string]interface{})

	program := make([]string, 0, len(programI))

	for _, programArgRaw := range programI {
		programArg, ok := programArgRaw.(string)

		if !ok || programArg == "" {
			continue
		}

		program = append(program, programArg)
	}

	if len(program) == 0 {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "External Program Missing",
				Detail:        "The data source was configured without a program to execute. Verify the configuration contains at least one non-empty value.",
				AttributePath: cty.GetAttrPath(programAttr),
			},
		}
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Query Handling Failed",
				Detail: "The data source received an unexpected error while attempting to parse the query. " +
					"This is always a bug in the external provider code and should be reported to the provider developers." +
					fmt.Sprintf("\n\nError: %s", err),
				AttributePath: cty.GetAttrPath(queryAttr),
			},
		}
	}

	// first element is assumed to be an executable command, possibly found
	// using the PATH environment variable.
	_, err = exec.LookPath(program[0])

	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "External Program Lookup Failed",
				Detail: `The data source received an unexpected error while attempting to find the program.

The program must be accessible according to the platform where Terraform is running.

If the expected program should be automatically found on the platform where Terraform is running, ensure that the program is in an expected directory. On Unix-based platforms, these directories are typically searched based on the '$PATH' environment variable. On Windows-based platforms, these directories are typically searched based on the '%PATH%' environment variable.

If the expected program is relative to the Terraform configuration, it is recommended that the program name includes the interpolated value of 'path.module' before the program name to ensure that it is compatible with varying module usage. For example: "${path.module}/my-program"

The program must also be executable according to the platform where Terraform is running. On Unix-based platforms, the file on the filesystem must have the executable bit set. On Windows-based platforms, no action is typically necessary.
` +
					fmt.Sprintf("\nPlatform: %s", runtime.GOOS) +
					fmt.Sprintf("\nProgram: %s", program[0]) +
					fmt.Sprintf("\nError: %s", err),
				AttributePath: cty.GetAttrPath(programAttr),
			},
		}
	}

	cmd := exec.CommandContext(ctx, program[0], program[1:]...)
	cmd.Dir = workingDir
	cmd.Stdin = bytes.NewReader(queryJson)

	if action != "" {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("TF_EXTERNAL_ACTION=%s", action))
	}

	tflog.Trace(ctx, "Executing external program", map[string]interface{}{"program": cmd.String()})

	resultJson, err := cmd.Output()

	tflog.Trace(ctx, "Executed external program", map[string]interface{}{"program": cmd.String(), "output": string(resultJson)})

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Stderr != nil && len(exitErr.Stderr) > 0 {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "External Program Execution Failed",
						Detail: "The data source received an unexpected error while attempting to execute the program." +
							fmt.Sprintf("\n\nProgram: %s", cmd.Path) +
							fmt.Sprintf("\nError Message: %s", string(exitErr.Stderr)) +
							fmt.Sprintf("\nState: %s", err),
						AttributePath: cty.GetAttrPath(programAttr),
					},
				}
			}

			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "External Program Execution Failed",
					Detail: "The data source received an unexpected error while attempting to execute the program.\n\n" +
						"The program was executed, however it returned no additional error messaging." +
						fmt.Sprintf("\n\nProgram: %s", cmd.Path) +
						fmt.Sprintf("\nState: %s", err),
					AttributePath: cty.GetAttrPath(programAttr),
				},
			}
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "External Program Execution Failed",
				Detail: "The data source received an unexpected error while attempting to execute the program." +
					fmt.Sprintf("\n\nProgram: %s", cmd.Path) +
					fmt.Sprintf("\nError: %s", err),
				AttributePath: cty.GetAttrPath(programAttr),
			},
		}
	}

	result := map[string]string{}
	err = json.Unmarshal(resultJson, &result)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Unexpected External Program Results",
				Detail: `The data source received unexpected results after executing the program.

Program output must be a JSON encoded map of string keys and string values.

If the error is unclear, the output can be viewed by enabling Terraform's logging at TRACE level. Terraform documentation on logging: https://www.terraform.io/internals/debugging
` +
					fmt.Sprintf("\nProgram: %s", cmd.Path) +
					fmt.Sprintf("\nResult Error: %s", err),
				AttributePath: cty.GetAttrPath(programAttr),
			},
		}
	}

	d.Set("result", result)

	d.SetId("-")
	return nil
}
