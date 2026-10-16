package container

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/synology-community/go-synology/pkg/api/docker"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/modifier"
)

// fakeProjects answers the Container Manager lookups ImportState makes.
// `list` leaves out the compose content, as ImportState must not rely on it.
type fakeProjects struct {
	docker.Api
	projects []docker.Project
}

func (f fakeProjects) ProjectGetByName(_ context.Context, name string) (*docker.Project, error) {
	for _, p := range f.projects {
		if p.Name == name {
			p.Content = ""
			return &p, nil
		}
	}
	return nil, docker.ProjectNotFoundError{}
}

func (f fakeProjects) ProjectGet(_ context.Context, id string) (*docker.Project, error) {
	for _, p := range f.projects {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, docker.ProjectNotFoundError{}
}

const vaultwardenCompose = `services:
  vaultwarden:
    image: vaultwarden/server:1.34.3
    environment:
      PUSH_ENABLED: "true"
    ports:
      - 5555:80
`

func TestProjectImportState(t *testing.T) {
	created := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	live := []docker.Project{
		{
			ID: "3f1c", Name: "vaultwarden", SharePath: "/docker/vaultwarden",
			Content: vaultwardenCompose, Status: "RUNNING", CreatedAt: created, UpdatedAt: created,
		},
		{ID: "9a0b", Name: "stopped", SharePath: "/projects/stopped", Content: "services: {}\n", Status: "STOPPED"},
	}

	tests := []struct {
		name      string
		importID  string
		wantErr   bool
		wantID    string
		wantName  string
		wantShare string
		wantBody  string
		wantRun   bool
		wantState string
	}{
		{name: "by name, running", importID: "vaultwarden", wantID: "3f1c", wantName: "vaultwarden",
			wantShare: "/docker/vaultwarden", wantBody: vaultwardenCompose, wantRun: true, wantState: "RUNNING"},
		{name: "by id, stopped", importID: "9a0b", wantID: "9a0b", wantName: "stopped",
			wantShare: "/projects/stopped", wantBody: "services: {}\n", wantRun: false, wantState: "STOPPED"},
		{name: "absent", importID: "nope", wantErr: true},
	}

	ctx := context.Background()
	r := &ProjectResource{client: fakeProjects{projects: live}}
	var sch resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sch)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := resource.ImportStateResponse{State: tfsdk.State{
				Schema: sch.Schema,
				Raw:    tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil),
			}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tt.importID}, &resp)
			if tt.wantErr {
				if !resp.Diagnostics.HasError() {
					t.Fatal("want an error for a project that does not exist")
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected error: %v", resp.Diagnostics)
			}

			var have models.ProjectResourceModel
			if d := resp.State.Get(ctx, &have); d.HasError() {
				t.Fatalf("reading state: %v", d)
			}
			for _, c := range []struct{ field, have, want string }{
				{"id", have.ID.ValueString(), tt.wantID},
				{"name", have.Name.ValueString(), tt.wantName},
				{"share_path", have.SharePath.ValueString(), tt.wantShare},
				{"content", have.Content.ValueString(), tt.wantBody},
				{"status", have.Status.ValueString(), tt.wantState},
			} {
				if c.have != c.want {
					t.Errorf("%s: have %q, want %q", c.field, c.have, c.want)
				}
			}
			if have.Run.ValueBool() != tt.wantRun {
				t.Errorf("run: have %v, want %v", have.Run.ValueBool(), tt.wantRun)
			}
			if !have.Services.IsNull() {
				t.Errorf("services: have %v, want null (the compose file is in content)", have.Services)
			}
		})
	}
}

func TestContentPlanModifierKeepsConfiguredContent(t *testing.T) {
	ctx := context.Background()
	var sch resource.SchemaResponse
	(&ProjectResource{}).Schema(ctx, resource.SchemaRequest{}, &sch)

	content := types.StringValue(vaultwardenCompose)
	plan := tfsdk.Plan{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)}
	for name, v := range map[string]attr.Value{"name": types.StringValue("vaultwarden"), "content": content} {
		if d := plan.SetAttribute(ctx, path.Root(name), v); d.HasError() {
			t.Fatalf("building plan: %v", d)
		}
	}
	cfg := tfsdk.Config{Schema: sch.Schema, Raw: plan.Raw}

	resp := planmodifier.StringResponse{PlanValue: content}
	modifier.UseSchemaForUnknownContent().PlanModifyString(ctx, planmodifier.StringRequest{
		Config: cfg, Plan: plan, ConfigValue: content, PlanValue: content,
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	if have := resp.PlanValue.ValueString(); have != vaultwardenCompose {
		t.Errorf("planned content: have %q, want the configured compose file", have)
	}
}
