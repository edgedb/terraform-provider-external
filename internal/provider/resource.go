package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceExternal() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "The `external` resource allows an external program implementing a specific protocol " +
			"(defined below) to act as a resource, exposing arbitrary data for use elsewhere in the Terraform " +
			"configuration.",

		CreateContext: resourceExternalCreate,
		ReadContext:   resourceExternalRead,
		UpdateContext: resourceExternalUpdate,
		DeleteContext: resourceExternalDelete,

		Schema: map[string]*schema.Schema{
			"program": {
				Description: "A list of strings, whose first element is the program to run and whose " +
					"subsequent elements are optional command line arguments to the program. Terraform does " +
					"not execute the program through a shell, so it is not necessary to escape shell " +
					"metacharacters nor add quotes around arguments containing spaces.",
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems: 1,
			},

			"program_destroy": {
				Description: "Same as *program*, but run on destroy (optional).",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems: 1,
			},

			"working_dir": {
				Description: "Working directory of the program. If not supplied, the program will run " +
					"in the current directory.",
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"working_dir_destroy": {
				Description: "Working directory of the program. If not supplied, the program will run " +
					"in the current directory.",
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"query": {
				Description: "A map of string values to pass to the external program as the query " +
					"arguments. If not supplied, the program will receive an empty object as its input.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"query_destroy": {
				Description: "A map of string values to pass to the external program as the query " +
					"arguments. If not supplied, the program will receive an empty object as its input.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"result": {
				Description: "A map of string values returned from the external program.",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceExternalCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ctx = context.WithValue(ctx, "tf_action", "create")
	return runProgram(ctx, "", d, meta)
}

func resourceExternalRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceExternalUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ctx = context.WithValue(ctx, "tf_action", "update")
	diag := runProgram(ctx, "destroy", d, meta)
	if diag.HasError() {
		return diag
	}
	return runProgram(ctx, "", d, meta)
}

func resourceExternalDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ctx = context.WithValue(ctx, "tf_action", "delete")
	return runProgram(ctx, "destroy", d, meta)
}
