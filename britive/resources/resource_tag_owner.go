package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	ownerEntityTypeUser = "User"
	ownerEntityTypeTag  = "Tag"
)

// ResourceTagOwner - Terraform Resource for Tag owners
type ResourceTagOwner struct {
	Resource     *schema.Resource
	helper       *ResourceTagOwnerHelper
	importHelper *imports.ImportHelper
}

// NewResourceTagOwner - Initializes new tag owner resource
func NewResourceTagOwner(importHelper *imports.ImportHelper) *ResourceTagOwner {
	rto := &ResourceTagOwner{
		helper:       NewResourceTagOwnerHelper(),
		importHelper: importHelper,
	}

	ownerEntitySchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The identifier of the owner entity",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The name of the owner entity",
			},
		},
	}

	rto.Resource = &schema.Resource{
		CreateContext: rto.resourceCreate,
		ReadContext:   rto.resourceRead,
		UpdateContext: rto.resourceUpdate,
		DeleteContext: rto.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rto.resourceStateImporter,
		},
		CustomizeDiff: validateOwnerBlocks,
		Schema: map[string]*schema.Schema{
			"tag_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the Britive tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"user": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "User owners of the tag",
				Elem:        ownerEntitySchema,
			},
			"tag": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Tag owners of the tag",
				Elem:        ownerEntitySchema,
			},
		},
	}

	return rto
}

// validateOwnerBlocks - CustomizeDiff validator that rejects any user or tag block
// where both id and name are set simultaneously.
func validateOwnerBlocks(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	for _, blockType := range []string{"user", "tag"} {
		if v, ok := d.GetOk(blockType); ok {
			for _, item := range v.(*schema.Set).List() {
				m := item.(map[string]interface{})
				id := m["id"].(string)
				name := m["name"].(string)
				if id == "" && name == "" {
					// skip empty blocks — TypeSet artifact during element removal
					continue
				}
				if id != "" && name != "" {
					return fmt.Errorf("%s block must specify either id or name, not both", blockType)
				}
			}
		}
	}
	return nil
}

//region Tag Owner Resource Context Operations

func (rto *ResourceTagOwner) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tagID := d.Get("tag_id").(string)
	log.Printf("[INFO] Creating tag owners for tag: %s", tagID)

	err := rto.helper.mapResourceToModel(d, m, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rto.helper.generateUniqueID(tagID))
	log.Printf("[INFO] Created tag owners for tag: %s", tagID)

	return rto.resourceRead(ctx, d, m)
}

func (rto *ResourceTagOwner) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	err := rto.helper.getAndMapModelToResource(d, m)
	if err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	return diags
}

func (rto *ResourceTagOwner) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tagID := d.Get("tag_id").(string)
	log.Printf("[INFO] Updating tag owners for tag: %s", tagID)

	err := rto.helper.mapResourceToModel(d, m, true)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated tag owners for tag: %s", tagID)
	return rto.resourceRead(ctx, d, m)
}

func (rto *ResourceTagOwner) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	c := m.(*britive.Client)

	tagID, err := rto.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting tag owners for tag: %s", tagID)

	tag, err := c.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.FromErr(err)
	}

	request := britive.TagWithOwners{
		TagID:       tagID,
		Name:        tag.Name,
		Description: tag.Description,
		Relationships: britive.TagOwnerRelationships{
			Owners: []britive.TagOwnerEntity{},
		},
	}

	if _, err = c.UpdateTagOwners(request); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted tag owners for tag: %s", tagID)
	d.SetId("")
	return diags
}

func (rto *ResourceTagOwner) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	importID := d.Id()

	if strings.TrimSpace(importID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("tag_id")
	}

	// Accept both "tags/{tagID}/owners" and plain "{tagID}"
	tagID := importID
	if strings.HasPrefix(importID, "tags/") && strings.HasSuffix(importID, "/owners") {
		tagID = strings.TrimSuffix(strings.TrimPrefix(importID, "tags/"), "/owners")
	}

	log.Printf("[INFO] Importing tag owners for tag id: %s", tagID)

	_, err := c.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("tag %s", tagID)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rto.helper.generateUniqueID(tagID))
	if err := d.Set("tag_id", tagID); err != nil {
		return nil, err
	}

	if err := rto.helper.getAndMapModelToResource(d, m); err != nil {
		return nil, err
	}

	log.Printf("[INFO] Imported tag owners for tag id: %s", tagID)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceTagOwnerHelper - helper functions for ResourceTagOwner
type ResourceTagOwnerHelper struct{}

// NewResourceTagOwnerHelper - Initializes new tag owner resource helper
func NewResourceTagOwnerHelper() *ResourceTagOwnerHelper {
	return &ResourceTagOwnerHelper{}
}

//region Tag Owner Resource helper functions

func (h *ResourceTagOwnerHelper) generateUniqueID(tagID string) string {
	return fmt.Sprintf("tags/%s/owners", tagID)
}

func (h *ResourceTagOwnerHelper) parseUniqueID(ID string) (tagID string, err error) {
	parts := strings.Split(ID, "/")
	if len(parts) < 3 {
		err = errs.NewInvalidResourceIDError("tag owner", ID)
		return
	}
	tagID = parts[1]
	return
}

// mapResourceToModel maps terraform resource data to the API request model and calls UpdateTagOwners.
func (h *ResourceTagOwnerHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, isUpdate bool) error {
	c := m.(*britive.Client)
	tagID := d.Get("tag_id").(string)

	tag, err := c.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("tag %s", tagID)
	}
	if err != nil {
		return err
	}

	owners := make([]britive.TagOwnerEntity, 0)

	for _, blockType := range []struct {
		key        string
		entityType string
	}{
		{"user", ownerEntityTypeUser},
		{"tag", ownerEntityTypeTag},
	} {
		if v, ok := d.GetOk(blockType.key); ok {
			for _, item := range v.(*schema.Set).List() {
				m := item.(map[string]interface{})
				id := m["id"].(string)
				name := m["name"].(string)
				// if id == "" && name == "" {
				// 	// skip empty blocks — TypeSet artifact during element removal
				// 	continue
				// }
				owner := britive.TagOwnerEntity{RelatedEntityType: blockType.entityType}
				if id != "" {
					owner.RelatedEntityID = id
				} else {
					owner.RelatedEntityName = name
				}
				owners = append(owners, owner)
			}
		}
	}

	request := britive.TagWithOwners{
		TagID:       tagID,
		Name:        tag.Name,
		Description: tag.Description,
		Relationships: britive.TagOwnerRelationships{
			Owners: owners,
		},
	}

	log.Printf("[INFO] Submitting tag owners for tag: %s", tagID)
	if _, err = c.UpdateTagOwners(request); err != nil {
		return err
	}

	return nil
}

// getAndMapModelToResource fetches tag owner data from the API and maps it to terraform resource state.
func (h *ResourceTagOwnerHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	tagID, err := h.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading tag owners for tag: %s", tagID)

	tagWithOwners, err := c.GetTagWithOwners(tagID)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received tag owners for tag: %#v", tagWithOwners)

	// Build state lookup maps keyed by whichever identifier the user configured
	// (id or name). The API is authoritative for what exists; state is authoritative
	// for how it was configured — same pattern as parameters in resource_resource_manager_resource_type.go.
	stateUserByKey := buildOwnerKeyMap(d.Get("user").(*schema.Set).List())
	stateTagByKey := buildOwnerKeyMap(d.Get("tag").(*schema.Set).List())

	var userList, tagList []map[string]interface{}

	for _, owner := range tagWithOwners.Relationships.Owners {
		switch owner.RelatedEntityType {
		case ownerEntityTypeUser:
			userList = append(userList, resolveOwnerEntry(owner, stateUserByKey))
		case ownerEntityTypeTag:
			tagList = append(tagList, resolveOwnerEntry(owner, stateTagByKey))
		}
	}

	if err := d.Set("tag_id", tagID); err != nil {
		return err
	}
	if err := d.Set("user", userList); err != nil {
		return err
	}
	if err := d.Set("tag", tagList); err != nil {
		return err
	}

	return nil
}

//endregion

// buildOwnerKeyMap builds a lookup map from a TypeSet list of owner state items.
// Because id and name are mutually exclusive, each item contributes exactly one key.
func buildOwnerKeyMap(stateItems []interface{}) map[string]map[string]interface{} {
	byKey := make(map[string]map[string]interface{}, len(stateItems))
	for _, stateItem := range stateItems {
		si := stateItem.(map[string]interface{})
		if id := si["id"].(string); id != "" {
			byKey[id] = si
		} else if name := si["name"].(string); name != "" {
			byKey[name] = si
		}
	}
	return byKey
}

// resolveOwnerEntry returns the state map for a single API owner, preserving the
// user-configured identifier (id or name). Falls back to id-only for external additions.
func resolveOwnerEntry(owner britive.TagOwnerEntity, stateByKey map[string]map[string]interface{}) map[string]interface{} {
	if si, ok := stateByKey[owner.RelatedEntityID]; ok {
		// return map[string]interface{}{"id": si["id"].(string), "name": si["name"].(string)}
		return map[string]interface{}{"id": si["id"].(string)}

	}
	if si, ok := stateByKey[owner.RelatedEntityName]; ok {
		return map[string]interface{}{"name": si["name"].(string)}
	}
	// External addition not in state — store with just id (stable identifier)
	return map[string]interface{}{"id": owner.RelatedEntityID, "name": ""}
}
