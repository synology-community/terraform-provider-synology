package models

type Share struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isdir"`
	Additional struct {
		Indexed        bool   `json:"indexed"`
		IsHybridShare  bool   `json:"is_hybrid_share"`
		IsWormShare    bool   `json:"is_worm_share"`
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
			IsACLMode       bool   `json:"is_acl_mode"`
			IsShareReadonly bool   `json:"is_share_readonly"`
			Posix           int    `json:"posix"`
			ShareRight      string `json:"share_right"`
		} `json:"perm"`
		RealPath  string `json:"real_path"`
		SyncShare bool   `json:"sync_share"`
		Time      struct {
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
}
