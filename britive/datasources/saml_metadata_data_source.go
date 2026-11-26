package datasources

import (
	"context"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &SAMLMetadataDataSource{}
	_ datasource.DataSourceWithConfigure = &SAMLMetadataDataSource{}
)

// NewSAMLMetadataDataSource is a helper function to simplify the provider implementation.
func NewSAMLMetadataDataSource() datasource.DataSource {
	return &SAMLMetadataDataSource{}
}

// SAMLMetadataDataSource is the data source implementation.
type SAMLMetadataDataSource struct {
	client *britive.Client
}

// SAMLMetadataDataSourceModel describes the data source data model.
type SAMLMetadataDataSourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Metadata                  types.String `tfsdk:"metadata"`
	Issuer                    types.String `tfsdk:"issuer"`
	SignInURL                 types.String `tfsdk:"sign_in_url"`
	SignOutURL                types.String `tfsdk:"sign_out_url"`
	CertificateExpirationDate types.String `tfsdk:"certificate_expiration_date"`
	SAMLCertificate           types.String `tfsdk:"saml_certificate"`
	OAuth2Certificate         types.String `tfsdk:"oauth2_certificate"`
}

// Metadata returns the data source type name.
func (d *SAMLMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_metadata"
}

// Schema defines the schema for the data source.
func (d *SAMLMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the Britive tenant's SAML identity provider metadata and certificates, for example to configure an AWS IAM SAML provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The identifier of the tenant's SAML configuration.",
			},
			"metadata": schema.StringAttribute{
				Computed:    true,
				Description: "The SAML 2.0 identity provider metadata XML, exactly as returned by Britive.",
			},
			"issuer": schema.StringAttribute{
				Computed:    true,
				Description: "The SAML identity provider entity ID.",
			},
			"sign_in_url": schema.StringAttribute{
				Computed:    true,
				Description: "The SAML sign-in URL.",
			},
			"sign_out_url": schema.StringAttribute{
				Computed:    true,
				Description: "The SAML sign-out URL.",
			},
			"certificate_expiration_date": schema.StringAttribute{
				Computed:    true,
				Description: "The expiration date of the SAML signing certificate.",
			},
			"saml_certificate": schema.StringAttribute{
				Computed:    true,
				Description: "The PEM-encoded SAML signing certificate, the same certificate embedded in `metadata`.",
			},
			"oauth2_certificate": schema.StringAttribute{
				Computed:    true,
				Description: "The tenant's PEM-encoded OAuth2 certificate.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *SAMLMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *SAMLMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SAMLMetadataDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := d.client.GetSAMLSettings()
	if err != nil && !errors.Is(err, britive.ErrNotFound) && !errors.Is(err, britive.ErrNoContent) {
		resp.Diagnostics.AddError(
			"Error Reading SAML Settings",
			fmt.Sprintf("Could not read the SAML settings: %s", err.Error()),
		)
		return
	}

	// A tenant has exactly one SAML configuration; anything else is unexpected.
	switch len(settings) {
	case 0:
		resp.Diagnostics.AddError(
			"SAML IdP Configuration Not Found",
			"The tenant has no SAML configuration.",
		)
		return
	case 1:
	default:
		resp.Diagnostics.AddError(
			"Multiple SAML IdP Configurations",
			fmt.Sprintf("Expected exactly one SAML configuration but the tenant returned %d (IDs: %s). Please report this issue to the provider developers.", len(settings), samlConfigurationIDs(settings)),
		)
		return
	}
	configuration := settings[0]

	metadata, err := d.client.GetSAMLMetadata(configuration.ID)
	if err != nil && !errors.Is(err, britive.ErrNoContent) {
		resp.Diagnostics.AddError(
			"Error Reading SAML IdP Metadata",
			fmt.Sprintf("Could not read the SAML metadata for configuration %d: %s", configuration.ID, err.Error()),
		)
		return
	}
	if strings.TrimSpace(metadata) == "" {
		resp.Diagnostics.AddError(
			"Empty SAML IdP Metadata",
			fmt.Sprintf("Britive returned empty SAML metadata for configuration %d.", configuration.ID),
		)
		return
	}
	if err := validateSAMLMetadata(metadata); err != nil {
		resp.Diagnostics.AddError(
			"Invalid SAML IdP Metadata",
			fmt.Sprintf("Britive returned SAML metadata for configuration %d that is not valid: %s", configuration.ID, err.Error()),
		)
		return
	}

	samlCertificate, err := d.client.GetSAMLCertificate(configuration.ID)
	if err != nil && !errors.Is(err, britive.ErrNoContent) {
		resp.Diagnostics.AddError(
			"Error Reading SAML Certificate",
			fmt.Sprintf("Could not read the SAML certificate for configuration %d: %s", configuration.ID, err.Error()),
		)
		return
	}
	if err := validatePEMCertificate(samlCertificate); err != nil {
		resp.Diagnostics.AddError(
			"Invalid SAML Certificate",
			fmt.Sprintf("Britive returned a SAML certificate for configuration %d that is not valid: %s", configuration.ID, err.Error()),
		)
		return
	}

	oauth2Certificate, err := d.client.GetOAuth2Certificate()
	if err != nil && !errors.Is(err, britive.ErrNoContent) {
		resp.Diagnostics.AddError(
			"Error Reading OAuth2 Certificate",
			fmt.Sprintf("Could not read the OAuth2 certificate: %s", err.Error()),
		)
		return
	}
	if err := validatePEMCertificate(oauth2Certificate); err != nil {
		resp.Diagnostics.AddError(
			"Invalid OAuth2 Certificate",
			fmt.Sprintf("Britive returned an OAuth2 certificate that is not valid: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(configuration.ID, 10))
	data.Metadata = types.StringValue(metadata)
	data.Issuer = types.StringValue(configuration.Issuer)
	data.SignInURL = types.StringValue(configuration.SignInURL)
	data.SignOutURL = types.StringValue(configuration.SignOutURL)
	data.CertificateExpirationDate = types.StringValue(configuration.X509CertExpirationDate)
	data.SAMLCertificate = types.StringValue(samlCertificate)
	data.OAuth2Certificate = types.StringValue(oauth2Certificate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// samlConfigurationIDs lists the configuration IDs for error messages.
func samlConfigurationIDs(settings []britive.SAMLSettings) string {
	ids := make([]string, 0, len(settings))
	for _, s := range settings {
		ids = append(ids, strconv.FormatInt(s.ID, 10))
	}
	return strings.Join(ids, ", ")
}

// validateSAMLMetadata checks that the document is well-formed XML with an EntityDescriptor root.
// The value stored in state stays the raw bytes returned by the API.
func validateSAMLMetadata(metadata string) error {
	decoder := xml.NewDecoder(strings.NewReader(metadata))
	rootFound := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("not well-formed XML: %w", err)
		}
		if start, ok := token.(xml.StartElement); ok && !rootFound {
			if start.Name.Local != "EntityDescriptor" {
				return fmt.Errorf("expected an EntityDescriptor root element, got %q", start.Name.Local)
			}
			rootFound = true
		}
	}
	if !rootFound {
		return errors.New("no root element found")
	}
	return nil
}

// validatePEMCertificate checks that the value is a PEM CERTIFICATE block.
// The value stored in state stays the raw text returned by the API.
func validatePEMCertificate(certificate string) error {
	if strings.TrimSpace(certificate) == "" {
		return errors.New("the response was empty")
	}
	block, _ := pem.Decode([]byte(certificate))
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("expected a PEM-encoded CERTIFICATE")
	}
	return nil
}
