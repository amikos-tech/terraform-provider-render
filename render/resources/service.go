package resources

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackall3n/render-go"
	"github.com/jackall3n/terraform-provider-render/render/models"
	"github.com/jackall3n/terraform-provider-render/render/types"
	"github.com/jackall3n/terraform-provider-render/render/utils"
	"net/http"
)

func ServiceResource() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	client  *render.ClientWithResponses
	context *types.Context
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	ctx, ok := req.ProviderData.(*types.Context)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *types.Context, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if ctx == nil {
		resp.Diagnostics.AddError(
			"Failed to initialize context",
			fmt.Sprintf("Expected *types.Context, got nil. Please report this issue to the provider developers."),
		)
		return
	}

	r.context = ctx
	r.client = ctx.Client
}

// Schema returns the schema information for a server resource.
func (r *serviceResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	disk := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"name":       schema.StringAttribute{Required: true},
			"mount_path": schema.StringAttribute{Required: true},
			"size_gb":    schema.Int64Attribute{Optional: true},
		},
	}

	resp.Schema = schema.Schema{
		Description: `Provider for service resource`,
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true},
			"name":        schema.StringAttribute{Required: true},
			"type":        schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"branch":      schema.StringAttribute{Optional: true, Computed: true},
			"auto_deploy": schema.BoolAttribute{Optional: true},
			"repo":        schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"owner":       schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},

			"web_service_details": schema.SingleNestedAttribute{
				Description: "Service details for `web_service` type services.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"env":                           schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
					"region":                        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
					"plan":                          schema.StringAttribute{Optional: true, Computed: true},
					"health_check_path":             schema.StringAttribute{Optional: true, Computed: true},
					"pull_request_previews_enabled": schema.BoolAttribute{Optional: true},
					"url":                           schema.StringAttribute{Computed: true},

					"native": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"build_command": schema.StringAttribute{Optional: true},
							"start_command": schema.StringAttribute{Optional: true},
						},
					},
					"disk": disk,
				},
			},

			"static_site_details": schema.SingleNestedAttribute{
				Description: "Service details for `static_site` type services.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"build_command":                 schema.StringAttribute{Optional: true},
					"publish_path":                  schema.StringAttribute{Optional: true},
					"pull_request_previews_enabled": schema.BoolAttribute{Optional: true},
					"url":                           schema.StringAttribute{Computed: true},
				},
			},

			"private_service_details": schema.SingleNestedAttribute{
				Description: "Service details for `private_service` type services.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"env":                           schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
					"region":                        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
					"plan":                          schema.StringAttribute{Optional: true, Computed: true},
					"pull_request_previews_enabled": schema.BoolAttribute{Optional: true},
					"url":                           schema.StringAttribute{Computed: true},
					"disk":                          disk,
				},
			},
		},
	}
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.Service

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ownerId, err := getOwner(r.context, plan)

	if err != nil {
		resp.Diagnostics.AddError("failed to get owner", err.Error())
		return
	}

	post, err := plan.ToServicePOST(ownerId)

	if err != nil {
		resp.Diagnostics.AddError("failed to convert to post", err.Error())
		return
	}

	tflog.Debug(ctx, "creating service", utils.ToJson(post))

	response, err := r.client.CreateServiceWithResponse(ctx, *post)

	if err != nil {
		resp.Diagnostics.AddError("failed to create service", err.Error())
		return
	}

	if response.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError("failed to create service", string(response.Body))
		return
	}

	s := response.JSON201.Service

	tflog.Debug(ctx, "Created service: "+response.Status(), map[string]interface{}{
		"s": s,
		"r": string(response.Body),
	})

	result := plan.FromResponse(*s)

	resp.State.Set(ctx, result)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.Service

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	s, err := r.client.GetServiceWithResponse(ctx, state.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading service",
			fmt.Sprintf("Could not read service %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := state.FromResponse(*s.JSON200)

	tflog.Trace(ctx, "read service", map[string]interface{}{
		"service_id": result.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state models.Service

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	patch, err := plan.ToServicePATCH()

	if err != nil {
		resp.Diagnostics.AddError("failed to convert to patch", err.Error())
		return
	}

	response, err := r.client.UpdateServiceWithResponse(ctx, state.ID.ValueString(), *patch)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating service",
			fmt.Sprintf(
				"Could not update service %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := plan.FromResponse(*response.JSON200)

	tflog.Debug(ctx, "updated service: "+response.Status(), map[string]interface{}{
		"service_id": result.ID.ValueString(),
		"service":    response.JSON200,
		"post":       patch,
		"json":       string(response.Body),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.Service
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteService(ctx, state.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting service",
			fmt.Sprintf(
				"Could not delete service %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "deleted service", map[string]interface{}{
		"service_id": state.ID.ValueString(),
	})
}

func getOwner(c *types.Context, plan models.Service) (string, error) {
	if plan.Owner.IsNull() || plan.Owner.ValueString() == "" {
		if c.Owner == nil {
			return "", fmt.Errorf("'owner' is required if a global email is not set")
		}

		return c.Owner.Id, nil
	}

	return plan.Owner.ValueString(), nil
}
