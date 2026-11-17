package file

import "github.com/abgeo/maroid/libs/pluginapi"

// The objects of one file under its key.
const (
	uploadObject    = "upload"
	thumbnailObject = "thumbnail.webp"
	displayObject   = "display.webp"
)

// objectKey answers the prefix of every object of one file. The prefix of the plugin
// leaves room for a file of the hub, and the owner keeps one scope apart from another.
func objectKey(pluginID string, scope pluginapi.FileScope, ownerID, fileID string) string {
	return "plugins/" + pluginID + "/" + string(scope) + "s/" + ownerID + "/" + fileID + "/"
}
