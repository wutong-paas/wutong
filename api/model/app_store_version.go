package model

type AppStoreVersionExportImageItem struct {
	Image    string `validate:"image|required" json:"image"`
	Username string `validate:"username" json:"username"`
	Password string `validate:"password" json:"password"`
}

type AppStoreVersionExportImageInfo struct {
	Images []AppStoreVersionExportImageItem `validate:"images|required" json:"images"`
}
