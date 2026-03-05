package validate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type ConstraintExclusiveFieldsValidator struct{}

func (v ConstraintExclusiveFieldsValidator) Description(ctx context.Context) string {
	return "Ensures that name is mutually exclusive with title, expression, and description"
}

func (v ConstraintExclusiveFieldsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ConstraintExclusiveFieldsValidator) ValidateResource(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var name, title, expression, description string

	req.Config.GetAttribute(ctx, path.Root("name"), &name)
	req.Config.GetAttribute(ctx, path.Root("title"), &title)
	req.Config.GetAttribute(ctx, path.Root("expression"), &expression)
	req.Config.GetAttribute(ctx, path.Root("description"), &description)

	nameSet := name != ""
	titleSet := title != ""
	expressionSet := expression != ""
	descriptionSet := description != ""

	if nameSet && (titleSet || expressionSet || descriptionSet) {
		resp.Diagnostics.AddError(
			"Invalid constraint configuration",
			"If `name` is set, then `title`, `expression`, and `description` cannot be set.",
		)
	}

	if !nameSet && !(titleSet || expressionSet || descriptionSet) {
		resp.Diagnostics.AddError(
			"Invalid constraint configuration",
			"You must set either `name` or at least one of `title`, `expression`, or `description`.",
		)
	}
}
