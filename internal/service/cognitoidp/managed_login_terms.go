// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package cognitoidp

import (
	"context"
	"fmt"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	awstypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkretry "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	intflex "github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_cognito_managed_login_terms", name="Managed Login Terms")
func newManagedLoginTermsResource(context.Context) (resource.ResourceWithConfigure, error) {
	r := &managedLoginTermsResource{}

	return r, nil
}

type managedLoginTermsResource struct {
	framework.ResourceWithModel[managedLoginTermsResourceModel]
}

func (r *managedLoginTermsResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrClientID: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
					stringvalidator.RegexMatches(
						regexache.MustCompile(`^[\w+]+$`),
						"must match [\\w+]+",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enforcement": schema.StringAttribute{
				CustomType: fwtypes.StringEnumType[awstypes.TermsEnforcementType](),
				Required:   true,
				Validators: []validator.String{
					stringvalidator.OneOf("NONE"),
				},
			},
			"links": schema.MapAttribute{
				CustomType: fwtypes.MapOfStringType,
				Required:   true,
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.SizeAtMost(12),
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexache.MustCompile(`^cognito:(default|english|french|spanish|german|bahasa-indonesia|italian|japanese|korean|portuguese-brazil|chinese-(simplified|traditional))$`),
							"invalid links key; see allowed Cognito language keys",
						),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.LengthBetween(1, 1024),
						stringvalidator.RegexMatches(
							regexache.MustCompile(`^[\p{L}\p{M}\p{S}\p{N}\p{P}]+$`),
							"invalid links value characters",
						),
					),
				},
			},
			"managed_login_terms_id": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexache.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[4][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`),
						"must be UUID v4",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"terms_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexache.MustCompile(`^(terms-of-use|privacy-policy)$`),
						"must be exactly \"terms-of-use\" or \"privacy-policy\"",
					),
				},
			},
			"terms_source": schema.StringAttribute{
				CustomType: fwtypes.StringEnumType[awstypes.TermsSourceType](),
				Required:   true,
				Validators: []validator.String{
					stringvalidator.OneOf("LINK"),
				},
			},
			"creation_date": schema.StringAttribute{
				CustomType: timetypes.RFC3339Type{},
				Computed:   true,
			},
			"last_modified_date": schema.StringAttribute{
				CustomType: timetypes.RFC3339Type{},
				Computed:   true,
			},
			names.AttrUserPoolID: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 55),
					stringvalidator.RegexMatches(
						regexache.MustCompile(`[\w-]+_[0-9a-zA-Z]+`),
						"must match [\\w-]+_[0-9a-zA-Z]+",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}
func (r *managedLoginTermsResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data managedLoginTermsResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().CognitoIDPClient(ctx)

	var input cognitoidentityprovider.CreateTermsInput
	response.Diagnostics.Append(fwflex.Expand(ctx, data, &input)...)
	if response.Diagnostics.HasError() {
		return
	}

	output, err := conn.CreateTerms(ctx, &input)
	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("creating Cognito Managed Login Terms (%s)", data.ClientID.ValueString()), err.Error())

		return
	}

	if output == nil || output.Terms == nil {
		response.Diagnostics.AddError("creating Cognito Managed Login Terms", tfresource.NewEmptyResultError(input).Error())

		return
	}

	response.Diagnostics.Append(fwflex.Flatten(ctx, output.Terms, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	data.ManagedLoginTermsID = fwflex.StringToFramework(ctx, output.Terms.TermsId)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *managedLoginTermsResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data managedLoginTermsResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().CognitoIDPClient(ctx)

	userPoolID, termsID := fwflex.StringValueFromFramework(ctx, data.UserPoolID), fwflex.StringValueFromFramework(ctx, data.ManagedLoginTermsID)
	terms, err := findManagedLoginTermsByTwoPartKey(ctx, conn, userPoolID, termsID)

	if retry.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)

		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("reading Cognito Managed Login Terms (%s)", termsID), err.Error())

		return
	}

	response.Diagnostics.Append(fwflex.Flatten(ctx, terms, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	data.ManagedLoginTermsID = fwflex.StringToFramework(ctx, terms.TermsId)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *managedLoginTermsResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state managedLoginTermsResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().CognitoIDPClient(ctx)

	var input cognitoidentityprovider.UpdateTermsInput
	response.Diagnostics.Append(fwflex.Expand(ctx, plan, &input)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Ensure TermsId is set from state; it's computed and not present in plan
	if state.ManagedLoginTermsID.IsNull() || state.ManagedLoginTermsID.IsUnknown() {
		response.Diagnostics.AddError("updating Cognito Managed Login Terms", "missing managed_login_terms_id in state")
		return
	}
	input.TermsId = aws.String(state.ManagedLoginTermsID.ValueString())

	output, err := conn.UpdateTerms(ctx, &input)

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("updating Cognito Managed Login Terms (%s)", state.ManagedLoginTermsID.ValueString()), err.Error())

		return
	}

	if output == nil || output.Terms == nil {
		response.Diagnostics.AddError("updating Cognito Managed Login Terms", tfresource.NewEmptyResultError(input).Error())

		return
	}

	response.Diagnostics.Append(fwflex.Flatten(ctx, output.Terms, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	plan.ManagedLoginTermsID = fwflex.StringToFramework(ctx, output.Terms.TermsId)

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *managedLoginTermsResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data managedLoginTermsResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().CognitoIDPClient(ctx)

	userPoolID, termsID := fwflex.StringValueFromFramework(ctx, data.UserPoolID), fwflex.StringValueFromFramework(ctx, data.ManagedLoginTermsID)
	tflog.Debug(ctx, "deleting Cognito Managed Login Terms", map[string]any{
		"managed_login_terms_id": termsID,
		names.AttrUserPoolID:     userPoolID,
	})
	input := cognitoidentityprovider.DeleteTermsInput{
		TermsId:    aws.String(termsID),
		UserPoolId: aws.String(userPoolID),
	}
	_, err := conn.DeleteTerms(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("deleting Cognito Managed Login Terms (%s)", termsID), err.Error())

		return
	}
}

func (r *managedLoginTermsResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	const (
		managedLoginTermsIDParts = 2
	)
	parts, err := intflex.ExpandResourceId(request.ID, managedLoginTermsIDParts, true)

	if err != nil {
		response.Diagnostics.Append(fwdiag.NewParsingResourceIDErrorDiagnostic(err))

		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root(names.AttrUserPoolID), parts[0])...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("managed_login_terms_id"), parts[1])...)
}

func (r *managedLoginTermsResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourceManagedLoginTermsLinksValidator{},
	}
}

func findManagedLoginTermsByTwoPartKey(ctx context.Context, conn *cognitoidentityprovider.Client, userPoolID, termsID string) (*awstypes.TermsType, error) {
	input := cognitoidentityprovider.DescribeTermsInput{
		TermsId:    aws.String(termsID),
		UserPoolId: aws.String(userPoolID),
	}

	output, err := conn.DescribeTerms(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return nil, &sdkretry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.Terms == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output.Terms, nil
}

type managedLoginTermsResourceModel struct {
	framework.WithRegionModel
	ClientID            types.String                                      `tfsdk:"client_id"`
	Enforcement         fwtypes.StringEnum[awstypes.TermsEnforcementType] `tfsdk:"enforcement"`
	Links               fwtypes.MapOfString                               `tfsdk:"links"`
	ManagedLoginTermsID types.String                                      `tfsdk:"managed_login_terms_id" autoflex:"TermsId"`
	CreationDate        timetypes.RFC3339                                 `tfsdk:"creation_date" autoflex:"CreationDate"`
	LastModifiedDate    timetypes.RFC3339                                 `tfsdk:"last_modified_date" autoflex:"LastModifiedDate"`
	TermsName           types.String                                      `tfsdk:"terms_name"`
	TermsSource         fwtypes.StringEnum[awstypes.TermsSourceType]      `tfsdk:"terms_source"`
	UserPoolID          types.String                                      `tfsdk:"user_pool_id"`
}

type resourceManagedLoginTermsLinksValidator struct{}

func (v resourceManagedLoginTermsLinksValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v resourceManagedLoginTermsLinksValidator) MarkdownDescription(context.Context) string {
	return "links must include a cognito:default entry"
}

func (v resourceManagedLoginTermsLinksValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config managedLoginTermsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Links.IsUnknown() || config.Links.IsNull() {
		return
	}

	links := make(map[string]string)
	// Allow unresolved values during plan to avoid conversion errors
	resp.Diagnostics.Append(config.Links.ElementsAs(ctx, &links, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, ok := links["cognito:default"]; !ok {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			path.Root("links"),
			v.MarkdownDescription(ctx),
			"missing cognito:default",
		))
	}
}
