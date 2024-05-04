package core

import "github.com/appkins/terraform-provider-synology/synology-go/api"

type CreateTaskRequest struct {
	api.BaseRequest

	SortBy     string   `synology:"sort_by"`
	FileType   string   `synology:"file_type"`
	CheckDir   bool     `synology:"check_dir"`
	Additional []string `synology:"additional"`
}

type CreateTaskResponse struct {
	api.BaseResponse

	Offset int `json:"offset"`

	Total int `json:"total"`
}

type ListTaskRequest struct {
	api.BaseRequest

	SortBy     string   `synology:"sort_by"`
	FileType   string   `synology:"file_type"`
	CheckDir   bool     `synology:"check_dir"`
	Additional []string `synology:"additional"`
	GoToPath   string   `synology:"goto_path"`
	FolderPath string   `synology:"folder_path"`
}

type ListTaskResponse struct {
	api.BaseResponse

	Offset int `json:"offset"`

	Tasks []struct {
		Name       string `json:"name"`
		Path       string `json:"path"`
		IsDir      bool   `json:"isdir"`
		Additional struct {
			Indexed        bool   `json:"indexed"`
			IsHybridTask   bool   `json:"is_hybrid_share"`
			IsWormTask     bool   `json:"is_worm_share"`
			MountPointType string `json:"mount_point_type"`
			Owner          struct {
				Group   string `json:"group"`
				GroupID int    `json:"gid"`
				User    string `json:"user"`
				UserID  int    `json:"uid"`
			} `json:"owner"`
			Perm struct {
				ACL struct {
					Append bool `json:"append"`
					Del    bool `json:"del"`
					Exec   bool `json:"exec"`
					Read   bool `json:"read"`
					Write  bool `json:"write"`
				} `json:"acl"`
				ACLEnable bool `json:"acl_enable"`
				AdvRight  struct {
					DisableDownload bool `json:"disable_download"`
					DisableList     bool `json:"disable_list"`
					DisableModify   bool `json:"disable_modify"`
				} `json:"adv_right"`
				IsACLMode      bool   `json:"is_acl_mode"`
				IsTaskReadonly bool   `json:"is_share_readonly"`
				Posix          int    `json:"posix"`
				TaskRight      string `json:"share_right"`
			} `json:"perm"`
			RealPath string `json:"real_path"`
			SyncTask bool   `json:"sync_share"`
			Time     struct {
				Atime  int `json:"atime"`
				Crtime int `json:"crtime"`
				Ctime  int `json:"ctime"`
				Mtime  int `json:"mtime"`
			} `json:"time"`
			VolumeStatus struct {
				Freespace  int64 `json:"freespace"`
				Readonly   bool  `json:"readonly"`
				Totalspace int64 `json:"totalspace"`
			} `json:"volume_status"`
			WormState int `json:"worm_state"`
		} `json:"additional"`
	} `json:"shares"`

	Total int `json:"total"`
}

var _ api.Request = (*ListTaskRequest)(nil)

func NewListTaskRequest(sortBy string, fileType string, checkDir bool, additional []string, goToPath string, folderPath string) *ListTaskRequest {

	if additional == nil {
		additional = []string{"real_path", "owner", "time", "perm", "mount_point_type", "sync_share", "volume_status", "indexed", "hybrid_share", "worm_share"}
	}
	if sortBy == "" {
		sortBy = "name"
	}
	return &ListTaskRequest{
		BaseRequest: api.BaseRequest{
			Version:   2,
			APIName:   "SYNO.FileStation.List",
			APIMethod: "list_share",
		},
		SortBy:     sortBy,
		FileType:   fileType,
		CheckDir:   checkDir,
		Additional: additional,
		GoToPath:   goToPath,
		FolderPath: folderPath,
	}
}

func (r ListTaskRequest) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{commonErrors}
}
