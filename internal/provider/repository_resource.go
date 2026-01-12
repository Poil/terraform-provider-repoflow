// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	// "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	// "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	// "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/fe80/go-repoflow/pkg/repoflow"

	"github.com/fe80/terraform-provider-repoflow/internal/factory"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RepositoryResource{}
var _ resource.ResourceWithImportState = &RepositoryResource{}

func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

// RepositoryResource defines the resource implementation.
type RepositoryResource struct {
	client *repoflow.Client
}

// RepositoryResourceModel describes the resource data model.
type RepositoryResourceModel struct {
	Name                              types.String `tfsdk:"name"`
	Id                                types.String `tfsdk:"id"`
	WorkspaceId                       types.String `tfsdk:"workspace"`
	PackageType                       types.String `tfsdk:"package_type"`
	RepositoryType                    types.String `tfsdk:"repository_type"`
	RepositoryId                      types.String `tfsdk:"repository_id"`
	RemoteRepositoryUrl               types.String `tfsdk:"remote_repository_url"`
	RemoteRepositoryUsername          types.String `tfsdk:"remote_repository_username"`
	RemoteRepositoryPassword          types.String `tfsdk:"remote_repository_password"`
	RemoteCacheEnabled                types.Bool   `tfsdk:"remote_cache_enabled"`
	FileCacheTimeTillRevalidation     types.Int64  `tfsdk:"file_cache_time_till_revalidation"`
	MetadataCacheTimeTillRevalidation types.Int64  `tfsdk:"metadata_cache_time_till_revalidation"`
	ChildRepositoryIds                types.List   `tfsdk:"child_repository_ids"`
	UploadLocalRepositoryId           types.String `tfsdk:"upload_local_repository_id"`
}

func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			//Required
			"name": schema.StringAttribute{
				MarkdownDescription: "Repository name to create.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: "Workspace used to create it (name or Id)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repository_type": schema.StringAttribute{
				MarkdownDescription: "Repository type stored by the repository.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("local", "remote", "virtual"),
				},
			},
			"package_type": schema.StringAttribute{
				MarkdownDescription: "Package type stored by the repository.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"cargo", "composer", "debian", "docker", "gems", "go", "helm",
						"maven", "npm", "nuget", "pypi", "rpm", "universal",
					),
				},
			},

			// Optional
			"remote_repository_url": schema.StringAttribute{
				MarkdownDescription: "URL of the remote repository (require for remote respository type).",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"remote_repository_username": schema.StringAttribute{
				MarkdownDescription: "Username for the remote repository.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"remote_repository_password": schema.StringAttribute{
				MarkdownDescription: "Password for the remote repository.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"remote_cache_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether caching is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"file_cache_time_till_revalidation": schema.Int64Attribute{
				MarkdownDescription: "Milliseconds before cached files require revalidation (null for indefinite caching).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"metadata_cache_time_till_revalidation": schema.Int64Attribute{
				MarkdownDescription: "Milliseconds before cached metadata requires revalidation (null for indefinite caching).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"child_repository_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of repositories included in the virtual repository. (require for virtual repository type)",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"upload_local_repository_id": schema.StringAttribute{
				MarkdownDescription: "ID of a local repository where uploads will be stored (must also be in child_repository_ids)..",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Computed attributes
			"repository_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository state identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*repoflow.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *repoflow.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *RepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RepositoryResourceModel
	var workspaceId string

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspace := data.WorkspaceId.ValueString()
	packageType := data.PackageType.ValueString()
	repositoryType := data.RepositoryType.ValueString()

	if ws, err := r.client.GetWorkspace(workspace); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get worksapce %s, got error: %s", workspace, err))
	} else {
		workspaceId = ws.Id
	}

	var err error
	var rp *repoflow.Repository

	switch repositoryType {
	case "local":
		opts := repoflow.RepositoryOptions{
			Name:        data.Name.ValueString(),
			PackageType: data.PackageType.ValueString(),
		}
		tflog.Debug(ctx, "create repository with option", map[string]interface{}{
			"opts": opts,
		})
		rp, err = r.client.CreateLocalRepository(workspace, opts)

	case "remote":
		if data.RemoteRepositoryUrl.IsNull() {
			resp.Diagnostics.AddError(
				"Missing parameter",
				"'remote_repository_url' is mandatory for remote repository type.",
			)
			return
		}

		opts := repoflow.RepositoryRemoteOptions{
			Name:                              data.Name.ValueString(),
			PackageType:                       data.PackageType.ValueString(),
			RemoteRepositoryUrl:               data.RemoteRepositoryUrl.ValueString(),
			RemoteRepositoryUsername:          data.RemoteRepositoryUsername.ValueString(),
			RemoteRepositoryPassword:          data.RemoteRepositoryPassword.ValueString(),
			IsRemoteCacheEnabled:              data.RemoteCacheEnabled.ValueBool(),
			FileCacheTimeTillRevalidation:     factory.Int64ToPtr(data.FileCacheTimeTillRevalidation),
			MetadataCacheTimeTillRevalidation: factory.Int64ToPtr(data.MetadataCacheTimeTillRevalidation),
		}
		tflog.Debug(ctx, "create repository with option", map[string]interface{}{
			"opts": opts,
		})
		rp, err = r.client.CreateRemoteRepository(workspace, opts)

	case "virtual":
		if data.ChildRepositoryIds.IsNull() {
			resp.Diagnostics.AddError(
				"Missing parameter",
				"`child_repository_ids` is required for virtual repository type.",
			)
			return
		}

		var childIds []string
		diags := data.ChildRepositoryIds.ElementsAs(ctx, &childIds, false)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() {
			return
		}

		uploadLocalRepositoryId := data.UploadLocalRepositoryId.ValueString()
		opts := repoflow.RepositoryVirtualOptions{
			Name:                    data.Name.ValueString(),
			PackageType:             data.PackageType.ValueString(),
			ChildRepositoryIds:      childIds,
			UploadLocalRepositoryId: uploadLocalRepositoryId,
		}
		tflog.Debug(ctx, "create repository with option", map[string]interface{}{
			"opts": opts,
		})
		rp, err = r.client.CreateVirtualRepository(workspace, opts)
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create repository, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(r.mapResponseToModel(ctx, &data, rp, workspaceId)...)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a repoflow repository resource", map[string]interface{}{
		"repository_id":   rp.Id,
		"package_type":    packageType,
		"repository_type": repositoryType,
		"workspace_id":    workspaceId,
		"id":              strings.Join([]string{workspaceId, rp.Id}, "/"),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspaceId := data.WorkspaceId.ValueString()
	repositoryId := data.RepositoryId.ValueString()

	rp, err := r.client.GetRepository(workspaceId, repositoryId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf(
			"Unable to get repository %s on workspaceId %s, got error: %s", repositoryId, workspaceId, err,
		))
		return
	}

	resp.Diagnostics.Append(r.mapResponseToModel(ctx, &data, rp, data.WorkspaceId.ValueString())...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "get a repoflow resource", map[string]interface{}{
		"id": rp.Id,
	})

	// Save updated data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspaceId := data.WorkspaceId.ValueString()
	repositoryId := data.RepositoryId.ValueString()

	rp, err := r.client.DeleteRepository(workspaceId, repositoryId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete repository, got error: %s", err))
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "deleted a repoflow resource", map[string]interface{}{
		"workspace": workspaceId,
		"id":        rp.RepositoryId,
	})
}

func (r *RepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data RepositoryResourceModel

	idParts := strings.Split(req.ID, "/")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Fail to import data",
			fmt.Sprintf("Id use format: workspaceId/repositoryId. You define: %q", req.ID),
		)
		return
	}

	var workspaceId string
	workspace := idParts[0]
	repository := idParts[1]

	if ws, err := r.client.GetWorkspace(workspace); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get worksapce %s, got error: %s", workspaceId, err))
	} else {
		workspaceId = ws.Id
	}

	rp, err := r.client.GetRepository(workspaceId, repository)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf(
			"Unable to import repository %s on workspaceId %s, got error: %s", repository, workspaceId, err,
		))
		return
	}

	resp.Diagnostics.Append(r.mapResponseToModel(ctx, &data, rp, workspaceId)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "import a repoflow repository resource", map[string]interface{}{
		"id": rp.Id,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) mapResponseToModel(ctx context.Context, data *RepositoryResourceModel, rp *repoflow.Repository, workspaceId string) diag.Diagnostics {
	var diags diag.Diagnostics

	// We save the state id with workspaceId/repositoryId
	data.Id = types.StringValue(strings.Join([]string{workspaceId, rp.Id}, "/"))
	// This is the real repository Id
	data.RepositoryId = types.StringValue(rp.Id)
	// We also save the Workspace Id in the state
	data.WorkspaceId = types.StringValue(workspaceId)

	// Default attributes
	data.Name = types.StringValue(rp.Name)
	if rp.RepositoryType != "" {
		data.PackageType = types.StringValue(rp.PackageType)
	}
	if rp.RepositoryType != "" {
		data.RepositoryType = types.StringValue(rp.RepositoryType)
	}

	// Remote attributes
	data.RemoteRepositoryUrl = types.StringPointerValue(rp.RemoteRepositoryUrl)
	data.RemoteRepositoryUsername = types.StringPointerValue(rp.RemoteRepositoryUsername)
	data.RemoteRepositoryPassword = types.StringPointerValue(rp.RemoteRepositoryPassword)
	data.RemoteCacheEnabled = types.BoolValue(rp.IsRemoteCacheEnabled)

	// Cache attributes utilisant ton package factory
	data.FileCacheTimeTillRevalidation = types.Int64PointerValue(factory.IntPtrToInt64Ptr(rp.FileCacheTimeTillRevalidation))
	data.MetadataCacheTimeTillRevalidation = types.Int64PointerValue(factory.IntPtrToInt64Ptr(rp.MetadataCacheTimeTillRevalidation))

	// Virtual attributes
	data.UploadLocalRepositoryId = types.StringPointerValue(rp.UploadLocalRepositoryId)

	// Handling ChildRepositories (conversion objets -> ids)
	if rp.ChildRepositories == nil {
		data.ChildRepositoryIds = types.ListNull(types.StringType)
	} else {
		ids := make([]string, len(rp.ChildRepositories))
		for i, child := range rp.ChildRepositories {
			ids[i] = child.Id
		}

		listValue, listDiags := types.ListValueFrom(ctx, types.StringType, ids)
		diags.Append(listDiags...)
		data.ChildRepositoryIds = listValue
	}

	return diags
}
