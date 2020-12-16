//  ASSETS PRE GO_EMBED
package assets

// _GITKEEP file
var _GITKEEP = []byte("")

// _VERSION file
var _VERSION = []byte("")

// _RSYNC file
var _RSYNC = []byte("")

// GetGITKEEP gets the file /.gitkeep from the stored data and returns the data.
func GetGITKEEP() []byte {
	return _GITKEEP
}

// GetVERSION gets the file /.version from the stored data and returns the data.
func GetVERSION() []byte {
	return _VERSION
}

// GetRSYNC gets the file /rsync from the stored data and returns the data.
func GetRSYNC() []byte {
	return _RSYNC
}
