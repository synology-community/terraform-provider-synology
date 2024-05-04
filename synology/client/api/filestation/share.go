package filestation

import "github.com/appkins/terraform-provider-synology/synology/client/api"

type CreateShareRequest struct {
	api.BaseRequest

	SortBy     string   `synology:"sort_by"`
	FileType   string   `synology:"file_type"`
	CheckDir   bool     `synology:"check_dir"`
	Additional []string `synology:"additional"`
}

type CreateShareResponse struct {
	api.BaseResponse

	Offset int `mapstructure:"offset" json:"offset"`

	Total int `mapstructure:"total" json:"total"`
}

func (r CreateShareResponse) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{commonErrors}
}

type ListShareRequest struct {
	api.BaseRequest

	SortBy     string   `synology:"sort_by"`
	FileType   string   `synology:"file_type"`
	CheckDir   bool     `synology:"check_dir"`
	Additional []string `synology:"additional"`
	GoToPath   string   `synology:"goto_path"`
	FolderPath string   `synology:"folder_path"`
}

type ListShareResponse struct {
	api.BaseResponse

	Offset int `mapstructure:"offset" json:"offset"`

	Shares []struct {
		Name       string `mapstructure:"name" json:"name"`
		Path       string `mapstructure:"path" json:"path"`
		IsDir      bool   `mapstructure:"isdir" json:"isdir"`
		Additional struct {
			Indexed        bool   `mapstructure:"indexed" json:"indexed"`
			IsHybridShare  bool   `mapstructure:"is_hybrid_share" json:"is_hybrid_share"`
			IsWormShare    bool   `mapstructure:"is_worm_share" json:"is_worm_share"`
			MountPointType string `mapstructure:"mount_point_type" json:"mount_point_type"`
			Owner          struct {
				Group   string `mapstructure:"group" json:"group"`
				GroupID int    `mapstructure:"gid" json:"gid"`
				User    string `mapstructure:"user" json:"user"`
				UserID  int    `mapstructure:"uid" json:"uid"`
			} `mapstructure:"owner" json:"owner"`
			Perm struct {
				ACL struct {
					Append bool `mapstructure:"append" json:"append"`
					Del    bool `mapstructure:"del" json:"del"`
					Exec   bool `mapstructure:"exec" json:"exec"`
					Read   bool `mapstructure:"read" json:"read"`
					Write  bool `mapstructure:"write" json:"write"`
				} `mapstructure:"acl" json:"acl"`
				ACLEnable bool `mapstructure:"acl_enable" json:"acl_enable"`
				AdvRight  struct {
					DisableDownload bool `mapstructure:"disable_download" json:"disable_download"`
					DisableList     bool `mapstructure:"disable_list" json:"disable_list"`
					DisableModify   bool `mapstructure:"disable_modify" json:"disable_modify"`
				} `mapstructure:"adv_right" json:"adv_right"`
				IsACLMode       bool   `mapstructure:"is_acl_mode" json:"is_acl_mode"`
				IsShareReadonly bool   `mapstructure:"is_share_readonly" json:"is_share_readonly"`
				Posix           int    `mapstructure:"posix" json:"posix"`
				ShareRight      string `mapstructure:"share_right" json:"share_right"`
			} `mapstructure:"perm" json:"perm"`
			RealPath  string `mapstructure:"real_path" json:"real_path"`
			SyncShare bool   `mapstructure:"sync_share" json:"sync_share"`
			Time      struct {
				Atime  int `mapstructure:"atime" json:"atime"`
				Crtime int `mapstructure:"crtime" json:"crtime"`
				Ctime  int `mapstructure:"ctime" json:"ctime"`
				Mtime  int `mapstructure:"mtime" json:"mtime"`
			} `mapstructure:"time" json:"time"`
			VolumeStatus struct {
				Freespace  int64 `mapstructure:"freespace" json:"freespace"`
				Readonly   bool  `mapstructure:"readonly" json:"readonly"`
				Totalspace int64 `mapstructure:"totalspace" json:"totalspace"`
			} `mapstructure:"volume_status" json:"volume_status"`
			WormState int `mapstructure:"worm_state" json:"worm_state"`
		} `mapstructure:"additional" json:"additional"`
	} `mapstructure:"shares" json:"shares"`

	Total int `mapstructure:"total" json:"total"`
}

var _ api.Request = (*ListShareRequest)(nil)

func NewListShareRequest(sortBy string, fileType string, checkDir bool, additional []string, goToPath string, folderPath string) *ListShareRequest {

	if additional == nil {
		additional = []string{"real_path", "owner", "time", "perm", "mount_point_type", "sync_share", "volume_status", "indexed", "hybrid_share", "worm_share"}
	}
	if sortBy == "" {
		sortBy = "name"
	}
	return &ListShareRequest{
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

func (r ListShareRequest) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{commonErrors}
}
