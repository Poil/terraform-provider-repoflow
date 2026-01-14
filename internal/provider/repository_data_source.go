package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/fe80/go-repoflow/pkg/repoflow"
	"github.com/fe80/terraform-provider-repoflow/internal/factory"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RepositoryDataSource{}

func NewRepositoryDataSource() datasource.DataSource {
	return &RepositoryDataSource{}
}

// ExampleDataSource defines the data source implementation.
type RepositoryDataSource struct {
	client *repoflow.Client
}

type RepositoryDataSourceModel struct {
	Name                              types.String `tfsdk:"name"`
	Id                                types.String `tfsdk:"id"`
	WorkspaceId                       types.String `tfsdk:"workspace"`
	PackageType                       types.String `tfsdk:"package_type"`
	RepositoryType                    types.String `tfsdk:"repository_type"`
	RepositoryId                      types.String `tfsdk:"repository_id"`
	RemoteRepositoryUrl               types.String `tfsdk:"remote_repository_url"`
	RemoteCacheEnabled                types.Bool   `tfsdk:"remote_cache_enabled"`
	FileCacheTimeTillRevalidation     types.Int64  `tfsdk:"file_cache_time_till_revalidation"`
	MetadataCacheTimeTillRevalidation types.Int64  `tfsdk:"metadata_cache_time_till_revalidation"`
	ChildRepositoryIds                types.List   `tfsdk:"child_repository_ids"`
	UploadLocalRepositoryId           types.String `tfsdk:"upload_local_repository_id"`
	Status                            types.String `tfsdk:"status"`
}

func (d *RepositoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *RepositoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Repository data source",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Repository name",
				Required:            true,
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: "Workspace used to create it (name or Id)",
				Required:            true,
			},
			"repository_type": schema.StringAttribute{
				MarkdownDescription: "Repository type stored by the repository.",
				Computed:            true,
			},
			"package_type": schema.StringAttribute{
				MarkdownDescription: "Package type stored by the repository.",
				Computed:            true,
			},
			"remote_repository_url": schema.StringAttribute{
				MarkdownDescription: "URL of the remote repository (require for remote respository type).",
				Computed:            true,
			},
			"remote_cache_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether caching is enabled.",
				Computed:            true,
			},
			"file_cache_time_till_revalidation": schema.Int64Attribute{
				MarkdownDescription: "Milliseconds before cached files require revalidation (null for indefinite caching).",
				Computed:            true,
			},
			"metadata_cache_time_till_revalidation": schema.Int64Attribute{
				MarkdownDescription: "Milliseconds before cached metadata requires revalidation (null for indefinite caching).",
				Computed:            true,
			},
			"child_repository_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of repositories included in the virtual repository. (require for virtual repository type)",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"upload_local_repository_id": schema.StringAttribute{
				MarkdownDescription: "ID of a local repository where uploads will be stored (must also be in child_repository_ids)..",
				Computed:            true,
			},
			"repository_id": schema.StringAttribute{
				MarkdownDescription: "Repository identifier",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the repository",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Repository identifier",
				Computed:            true,
			},
		},
	}
}

func (d *RepositoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*repoflow.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *RepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RepositoryDataSourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspace := data.WorkspaceId.ValueString()
	repository := data.Name.ValueString()

	var workspaceId string
	if ws, err := d.client.GetWorkspace(workspace); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get worksapce %s, got error: %s", workspaceId, err))
	} else {
		workspaceId = ws.Id
	}

	rp, err := d.client.GetRepository(workspaceId, repository)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf(
			"Unable to read repository %s on workspaceId %s, got error: %s", repository, workspaceId, err,
		))
		return
	}

	// We save the state id with workspaceId/repositoryId
	data.Id = types.StringValue(strings.Join([]string{workspaceId, rp.Id}, "/"))
	// This is the real repository Id
	data.RepositoryId = types.StringValue(rp.Id)
	// We also save the Workspace Id in the state
	data.WorkspaceId = types.StringValue(workspaceId)

	// Default attributes
	data.Name = types.StringValue(rp.Name)
	data.Status = types.StringValue(rp.Status)
	if rp.RepositoryType != "" {
		data.PackageType = types.StringValue(rp.PackageType)
	}
	if rp.RepositoryType != "" {
		data.RepositoryType = types.StringValue(rp.RepositoryType)
	}

	// Remote attributes
	data.RemoteRepositoryUrl = types.StringPointerValue(rp.RemoteRepositoryUrl)
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
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.ChildRepositoryIds = listValue
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read repository data", map[string]interface{}{
		"name":      repository,
		"id":        rp.Id,
		"workspace": workspaceId,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
