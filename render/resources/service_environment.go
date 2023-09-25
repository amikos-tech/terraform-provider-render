package resources

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackall3n/render-go"
	"github.com/jackall3n/terraform-provider-render/render/models"
	"github.com/jackall3n/terraform-provider-render/render/types"
	"github.com/jackall3n/terraform-provider-render/render/utils"
	"net/http"
)

func ServiceEnvironmentResource() resource.Resource {
	return &serviceEnvironmentResource{}
}

type serviceEnvironmentResource struct {
	client  *render.ClientWithResponses
	context *types.Context
}

func (r *serviceEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_environment"
}

func (r *serviceEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	ctx, ok := req.ProviderData.(*types.Context)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.context = ctx
	r.client = ctx.Client
}

// Schema returns the schema information for a server resource.
func (r *serviceEnvironmentResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provider for service environment resource`,
		Attributes: map[string]schema.Attribute{
			"service": schema.StringAttribute{Required: true},

			"variables": schema.ListNestedAttribute{
				Description: "Service environment variable",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":       schema.StringAttribute{Required: true},
						"value":     schema.StringAttribute{Optional: true},
						"generated": schema.BoolAttribute{Optional: true},
					},
				},
			},
		},
	}
}

func (r *serviceEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.ServiceEnvironment

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var variables []render.EnvVarsPATCH_Item

	for _, i := range plan.Variables {
		item, err := i.ToEnvVarsItemPATCH()

		if err != nil {
			resp.Diagnostics.AddError("failed to convert to item", err.Error())
			return
		}

		variables = append(variables, *item)
	}

	tflog.Debug(ctx, "setting service variables", utils.ToJson(map[string]interface{}{
		"variables": variables,
	}))

	response, err := r.client.UpdateEnvVarsForServiceWithResponse(ctx, plan.Service.ValueString(), variables)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update service variables",
			fmt.Sprintf("Could not update service variables %s, unexpected error: %s",
				err.Error(),
				err,
			),
		)
		return
	}

	if response.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to update service variables",
			fmt.Sprintf("Could not update service variables %s, unexpected error: %s",
				response.Status(),
				string(response.Body),
			),
		)
		return
	}

	s := response.JSON200

	tflog.Debug(ctx, "Update service env vars: "+response.Status(), map[string]interface{}{
		"s": s,
		"r": string(response.Body),
	})

	//result := plan.FromResponse(*s)

	resp.State.Set(ctx, plan)
}

func (r *serviceEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.ServiceEnvironment

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "read service variables", map[string]interface{}{
		"service": state.Service.ValueString(),
	})

	response, err := r.client.GetEnvVarsForServiceWithResponse(ctx, state.Service.ValueString(), &render.GetEnvVarsForServiceParams{})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading service variables",
			fmt.Sprintf("Could not read service variables %s, unexpected error: %s",
				state.Service.ValueString(),
				err,
			),
		)

		return
	}

	if response.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Error reading service variables",
			fmt.Sprintf("Could not read service variables %s %s, unexpected error: %s",
				response.Status(),
				state.Service.ValueString(),
				string(response.Body),
			),
		)

		return
	}

	var variables []models.ServiceEnvironmentVariable

	for _, item := range *response.JSON200 {
		variables = append(variables, models.ServiceEnvironmentVariable{}.FromResponse(*item.EnvVar))
	}

	tflog.Trace(ctx, "read service variables", map[string]interface{}{
		"variables": variables,
	})

	diags = resp.State.Set(ctx, state.FromResponse(variables))

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *serviceEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.ServiceEnvironment

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	variables := render.UpdateEnvVarsForServiceJSONRequestBody{}

	for _, i := range plan.Variables {
		item, err := i.ToEnvVarsItemPATCH()

		if err != nil {
			resp.Diagnostics.AddError("failed to convert to item", err.Error())
			return
		}

		variables = append(variables, *item)
	}

	tflog.Debug(ctx, "setting service variables", utils.ToJson(map[string]interface{}{
		"variables": variables,
	}))

	response, err := r.client.UpdateEnvVarsForServiceWithResponse(ctx, plan.Service.ValueString(), variables)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update service variables",
			fmt.Sprintf("Could not update service [%s] variables %s, unexpected error: %s",
				plan.Service.ValueString(),
				err.Error(),
				err,
			),
		)
		return
	}

	if response.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to update service variables",
			fmt.Sprintf("Could not update service [%s] variables\nresponse: %s %s\nNote: If you're trying to use 'generated' and the response says 'invalid JSON', this is an issue with the render api, not this provider.",
				plan.Service.ValueString(),
				response.Status(),
				string(response.Body),
			),
		)
		return
	}

	s := response.JSON200

	tflog.Debug(ctx, "Update service env vars: "+response.Status(), map[string]interface{}{
		"s": s,
		"r": string(response.Body),
	})

	resp.State.Set(ctx, plan)
}

func (r *serviceEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.ServiceEnvironment

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var variables []render.EnvVarsPATCH_Item

	tflog.Debug(ctx, "deleting service variables")

	response, err := r.client.UpdateEnvVarsForServiceWithResponse(ctx, state.Service.ValueString(), variables)

	if err != nil {
		resp.Diagnostics.AddError("failed to update service variables1", err.Error())
		return
	}

	if response.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("failed to update service variables", string(response.Body))
		return
	}

	s := response.JSON200

	tflog.Debug(ctx, "Update service env vars: "+response.Status(), map[string]interface{}{
		"s": s,
		"r": string(response.Body),
	})

	//result := plan.FromResponse(*s)

	//resp.State.Set(ctx, state)
}
