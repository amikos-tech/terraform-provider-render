package models

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jackall3n/render-go"
	"github.com/jackall3n/terraform-provider-render/render/utils"
)

type Service struct {
	ID                    types.String           `tfsdk:"id"`
	Name                  types.String           `tfsdk:"name"`
	Type                  types.String           `tfsdk:"type"`
	Repo                  types.String           `tfsdk:"repo"`
	Branch                types.String           `tfsdk:"branch"`
	Owner                 types.String           `tfsdk:"owner"`
	AutoDeploy            types.Bool             `tfsdk:"auto_deploy"`
	WebServiceDetails     *WebServiceDetails     `tfsdk:"web_service_details"`
	StaticSiteDetails     *StaticSiteDetails     `tfsdk:"static_site_details"`
	PrivateServiceDetails *PrivateServiceDetails `tfsdk:"private_service_details"`
}

type WebServiceDetails struct {
	Env                        types.String             `tfsdk:"env"`
	Region                     types.String             `tfsdk:"region"`
	Plan                       types.String             `tfsdk:"plan"`
	PullRequestPreviewsEnabled types.Bool               `tfsdk:"pull_request_previews_enabled"`
	HealthCheckPath            types.String             `tfsdk:"health_check_path"`
	Native                     *WebServiceDetailsNative `tfsdk:"native"`
	Url                        types.String             `tfsdk:"url"`
	Disk                       *Disk                    `tfsdk:"disk"`
}

type WebServiceDetailsNative struct {
	BuildCommand types.String `tfsdk:"build_command"`
	StartCommand types.String `tfsdk:"start_command"`
}

type StaticSiteDetails struct {
	BuildCommand               types.String `tfsdk:"build_command"`
	PublishPath                types.String `tfsdk:"publish_path"`
	PullRequestPreviewsEnabled types.Bool   `tfsdk:"pull_request_previews_enabled"`
	Url                        types.String `tfsdk:"url"`
}

type PrivateServiceDetails struct {
	Env                        types.String `tfsdk:"env"`
	Region                     types.String `tfsdk:"region"`
	Plan                       types.String `tfsdk:"plan"`
	PullRequestPreviewsEnabled types.Bool   `tfsdk:"pull_request_previews_enabled"`
	Url                        types.String `tfsdk:"url"`
	Disk                       *Disk        `tfsdk:"disk"`
}

type Disk struct {
	Name      types.String `tfsdk:"name"`
	MountPath types.String `tfsdk:"mount_path"`
	SizeGB    types.Int64  `tfsdk:"size_gb"`
}

func (s Service) FromResponse(response render.Service) Service {
	serviceType := *response.Type

	service := Service{
		ID:     fromStringOptional(response.Id),
		Name:   fromStringOptional(response.Name),
		Type:   fromServiceType(response.Type),
		Repo:   fromStringOptional(response.Repo),
		Branch: fromStringOptional(response.Branch),
		Owner:  fromStringOptional(response.OwnerId),
	}

	if serviceType == render.WebService {
		details, _ := response.ServiceDetails.AsWebServiceDetails()

		service.WebServiceDetails = &WebServiceDetails{
			Region:          fromRegion(details.Region),
			Env:             fromServiceEnv(details.Env),
			Plan:            fromStringOptional(details.Plan),
			HealthCheckPath: fromStringOptional(details.HealthCheckPath),
			Url:             fromStringOptional(details.Url),
		}

		native, err := details.EnvSpecificDetails.AsNativeEnvironmentDetails()

		if err == nil {
			service.WebServiceDetails.Native = &WebServiceDetailsNative{
				BuildCommand: fromStringOptional(native.BuildCommand),
				StartCommand: fromStringOptional(native.StartCommand),
			}
		}
		if details.Disk != nil {
			service.WebServiceDetails.Disk = &Disk{
				Name: fromStringOptional(details.Disk.Name),

				// Hack because the OpenAPI doesn't specify these fields as return.. I should check this
				MountPath: s.WebServiceDetails.Disk.MountPath,
				SizeGB:    s.WebServiceDetails.Disk.SizeGB,
			}
		}
	}

	if serviceType == render.PrivateService {
		details, _ := response.ServiceDetails.AsPrivateServiceDetails()

		service.PrivateServiceDetails = &PrivateServiceDetails{
			Region: fromRegion(details.Region),
			Env:    fromServiceEnv(details.Env),
			Plan:   fromStringOptional(details.Plan),
			Url:    fromStringOptional(details.Url),
		}

		if details.Disk != nil {
			service.PrivateServiceDetails.Disk = &Disk{
				Name: fromStringOptional(details.Disk.Name),

				// Hack because the OpenAPI doesn't specify these fields as return.. I should check this
				MountPath: s.PrivateServiceDetails.Disk.MountPath,
				SizeGB:    s.PrivateServiceDetails.Disk.SizeGB,
			}
		}
	}

	if serviceType == render.StaticSite {
		details, _ := response.ServiceDetails.AsStaticSiteDetails()

		service.StaticSiteDetails = &StaticSiteDetails{
			BuildCommand: fromStringOptional(details.BuildCommand),
			PublishPath:  fromStringOptional(details.PublishPath),
			Url:          fromStringOptional(details.Url),
		}
	}

	return service
}

func (s Service) ToServicePOST(ownerId string) (*render.ServicePOST, error) {
	serviceType := render.ServiceType(s.Type.ValueString())

	service := render.ServicePOST{
		Type:    serviceType,
		Name:    s.Name.ValueString(),
		Repo:    s.Repo.ValueString(),
		Branch:  stringOptionalNil(s.Branch),
		OwnerId: ownerId,
	}

	serviceDetails := render.ServicePOST_ServiceDetails{}

	if serviceType == render.WebService || s.WebServiceDetails != nil {
		if s.WebServiceDetails == nil {
			return nil, fmt.Errorf("'web_service_details' is required for services of type 'web_service'")
		}

		if serviceType != render.WebService {
			return nil, fmt.Errorf("'web_service_details' can only be used for services of type 'web_service'")
		}

		details := render.WebServiceDetailsPOST{}
		mapped, err := toWebServiceDetails(s.WebServiceDetails)

		if err != nil {
			return nil, err
		}

		if utils.Struct(mapped, &details) != nil {
			return nil, err
		}

		if serviceDetails.FromWebServiceDetailsPOST(details) != nil {
			return nil, err
		}
	}

	if serviceType == render.StaticSite || s.StaticSiteDetails != nil {
		if s.StaticSiteDetails == nil {
			return nil, fmt.Errorf("'static_site_details' is required for services of type 'static_site'")
		}

		if serviceType != render.StaticSite {
			return nil, fmt.Errorf("'static_site_details' can only be used for services of type 'static_site'")
		}

		details := render.StaticSiteDetailsPOST{}
		mapped, err := toStaticSiteDetails(s.StaticSiteDetails)

		if err != nil {
			return nil, err
		}

		if utils.Struct(mapped, &details) != nil {
			return nil, err
		}

		if serviceDetails.FromStaticSiteDetailsPOST(details) != nil {
			return nil, err
		}
	}

	if serviceType == render.PrivateService || s.PrivateServiceDetails != nil {
		if s.PrivateServiceDetails == nil {
			return nil, fmt.Errorf("'private_service_details' is required for services of type 'private_service'")
		}

		if serviceType != render.PrivateService {
			return nil, fmt.Errorf("'private_service_details' can only be used for services of type 'private_service'")
		}

		details := render.PrivateServiceDetailsPOST{}
		mapped, err := toPrivateServiceDetails(s.PrivateServiceDetails)

		if err != nil {
			return nil, err
		}

		if utils.Struct(mapped, &details) != nil {
			return nil, err
		}

		if serviceDetails.FromPrivateServiceDetailsPOST(details) != nil {
			return nil, err
		}
	}

	service.ServiceDetails = &serviceDetails

	return &service, nil
}

func (s Service) ToServicePATCH() (*render.ServicePATCH, error) {
	serviceType := render.ServiceType(s.Type.ValueString())

	service := render.ServicePATCH{
		Name:   stringOptional(s.Name),
		Branch: stringOptionalNil(s.Branch),
	}

	serviceDetails := render.ServicePATCH_ServiceDetails{}

	if serviceType == render.WebService || s.WebServiceDetails != nil {
		if s.WebServiceDetails == nil {
			return nil, fmt.Errorf("'web_service_details' is required for services of type 'web_service'")
		}

		if serviceType != render.WebService {
			return nil, fmt.Errorf("'web_service_details' can only be used for services of type 'web_service'")
		}

		details := render.WebServiceDetailsPATCH{}
		mapped, err := toWebServiceDetails(s.WebServiceDetails)

		if err != nil {
			return nil, err
		}

		if utils.Struct(mapped, &details) != nil {
			return nil, err
		}

		if serviceDetails.FromWebServiceDetailsPATCH(details) != nil {
			return nil, err
		}
	}

	if serviceType == render.StaticSite || s.StaticSiteDetails != nil {
		if s.StaticSiteDetails == nil {
			return nil, fmt.Errorf("'static_site_details' is required for services of type 'static_site'")
		}

		if serviceType != render.StaticSite {
			return nil, fmt.Errorf("'static_site_details' can only be used for services of type 'static_site'")
		}

		details := render.StaticSiteDetailsPATCH{}
		mapped, err := toStaticSiteDetails(s.StaticSiteDetails)

		if err != nil {
			return nil, err
		}

		if utils.Struct(mapped, &details) != nil {
			return nil, err
		}

		if serviceDetails.FromStaticSiteDetailsPATCH(details) != nil {
			return nil, err
		}
	}

	if serviceType == render.PrivateService || s.PrivateServiceDetails != nil {
		if s.PrivateServiceDetails == nil {
			return nil, fmt.Errorf("'private_service_details' is required for services of type 'private_service'")
		}

		if serviceType != render.PrivateService {
			return nil, fmt.Errorf("'private_service_details' can only be used for services of type 'private_service'")
		}

		details := render.PrivateServiceDetailsPATCH{}
		mapped, err := toPrivateServiceDetails(s.PrivateServiceDetails)

		if err != nil {
			return nil, err
		}

		if utils.Struct(mapped, &details) != nil {
			return nil, err
		}

		if serviceDetails.FromPrivateServiceDetailsPATCH(details) != nil {
			return nil, err
		}
	}

	service.ServiceDetails = &serviceDetails

	return &service, nil
}

func toWebServiceDetails(webServiceDetails *WebServiceDetails) (map[string]interface{}, error) {
	details := map[string]interface{}{
		"region":          stringOptionalNil(webServiceDetails.Region),
		"env":             stringOptional(webServiceDetails.Env),
		"plan":            stringOptionalNil(webServiceDetails.Plan),
		"healthCheckPath": stringOptional(webServiceDetails.HealthCheckPath),
	}

	if webServiceDetails.Native != nil {
		native := map[string]interface{}{
			"buildCommand": webServiceDetails.Native.BuildCommand.ValueString(),
			"startCommand": webServiceDetails.Native.StartCommand.ValueString(),
		}

		details["envSpecificDetails"] = native
	}

	if webServiceDetails.Disk != nil {
		details["disk"] = toDisk(webServiceDetails.Disk)
	}

	return details, nil
}

func toPrivateServiceDetails(serviceDetails *PrivateServiceDetails) (map[string]interface{}, error) {
	details := map[string]interface{}{
		"region": stringOptionalNil(serviceDetails.Region),
		"env":    stringOptional(serviceDetails.Env),
		"plan":   stringOptionalNil(serviceDetails.Plan),
	}

	if serviceDetails.Disk != nil {
		details["disk"] = toDisk(serviceDetails.Disk)
	}

	return details, nil
}

func toStaticSiteDetails(staticSiteDetails *StaticSiteDetails) (map[string]interface{}, error) {
	details := map[string]interface{}{
		"buildCommand": staticSiteDetails.BuildCommand.ValueString(),
		"publishPath":  staticSiteDetails.PublishPath.ValueString(),
	}

	return details, nil
}

func toDisk(d *Disk) map[string]interface{} {
	disk := map[string]interface{}{
		"name":      d.Name.ValueString(),
		"mountPath": d.MountPath.ValueString(),
		"sizeGB":    int64Optional(d.SizeGB),
	}

	return disk
}

func stringOptional(str types.String) *string {
	if str.IsNull() {
		return nil
	}

	value := str.ValueString()

	return &value
}

func stringOptionalNil(str types.String) *string {
	if str.IsNull() || str.ValueString() == "" {
		return nil
	}

	value := str.ValueString()

	return &value
}

func int64Optional(num types.Int64) *int64 {
	if num.IsNull() {
		return nil
	}

	value := num.ValueInt64()

	return &value
}

func fromIntOptional(num *int) types.Int64 {
	if num == nil {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*num))
}

func fromStringOptional(str *string) types.String {
	if str == nil {
		return types.StringNull()
	}

	return types.StringValue(*str)
}

func fromServiceType(t *render.ServiceType) types.String {
	if t == nil {
		return types.StringNull()
	}

	return types.StringValue(string(*t))
}

func fromServiceEnv(e *render.ServiceEnv) types.String {
	if e == nil {
		return types.StringNull()
	}

	return types.StringValue(string(*e))
}

func fromRegion(r *render.Region) types.String {
	if r == nil {
		return types.StringNull()
	}

	return types.StringValue(string(*r))
}
