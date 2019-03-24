//go:generate go install -v github.com/josephspurrier/goversioninfo/cmd/goversioninfo
//go:generate goversioninfo -icon=res/papp.ico
package main

import (
	"os"

	. "github.com/portapps/portapps"
	"github.com/portapps/portapps/pkg/registry"
	"github.com/portapps/portapps/pkg/utl"
)

type config struct {
	Verbose string `yaml:"verbose" mapstructure:"verbose"`
}

var (
	app *App
	cfg *config
)

func init() {
	var err error

	// Default config
	cfg = &config{
		Verbose: "1",
	}

	// Init app
	if app, err = NewWithCfg("vlc-portable", "VLC", cfg); err != nil {
		Log.Fatal().Err(err).Msg("Cannot initialize application. See log file for more info.")
	}
}

func main() {
	utl.CreateFolder(app.DataPath)
	app.Process = utl.PathJoin(app.AppPath, "vlc.exe")
	app.Args = []string{
		"--vlm-conf=" + utl.PathJoin(app.DataPath, "vlcrc"),
		"--config=" + utl.PathJoin(app.DataPath, "vlcrc"),
		"--no-plugins-cache",
		"--no-qt-updates-notif",
	}

	// VLC paths
	vlcRoamingPath := utl.PathJoin(utl.RoamingPath(), "vlc")
	vlcTmpPath := utl.CreateFolder(app.AppPath, "tmp")

	// Set env vars
	utl.OverrideEnv("VLC_PLUGIN_PATH", utl.CreateFolder(app.DataPath, "plugins"))
	utl.OverrideEnv("VLC_VERBOSE", cfg.Verbose)
	utl.OverrideEnv("TEMP", vlcTmpPath)

	// VLC volatile files
	dataDvdcssPath := utl.PathJoin(app.DataPath, "dvdcss")
	dataMlXspf := utl.PathJoin(app.DataPath, "ml.xspf")
	dataVlcQtInterface := utl.PathJoin(app.DataPath, "vlc-qt-interface.ini")
	roamingDvdcssPath := utl.PathJoin(utl.RoamingPath(), "dvdcss")
	roamingMlXspf := utl.PathJoin(vlcRoamingPath, "ml.xspf")
	roamingVlcQtInterface := utl.PathJoin(vlcRoamingPath, "vlc-qt-interface.ini")

	// Copy existing files from data to roaming folder for the current user
	utl.CreateFolder(vlcRoamingPath)
	if _, err := os.Stat(dataMlXspf); err == nil {
		utl.CopyFile(dataMlXspf, roamingMlXspf)
	}
	if _, err := os.Stat(dataVlcQtInterface); err == nil {
		utl.CopyFile(dataVlcQtInterface, roamingVlcQtInterface)
	}

	// Handle reg key
	regsPath := utl.CreateFolder(app.RootPath, "reg")
	regFile := utl.PathJoin(regsPath, "VLC.reg")
	regKey := registry.ExportImport{
		Key:  `HKCU\Software\VideoLAN\VLC`,
		Arch: "32",
		File: regFile,
	}
	if err := registry.ImportKey(regKey); err != nil {
		Log.Warn().Err(err).Msg("Cannot import registry key")
	}

	// On exit
	defer func() {
		// Copy back to data
		if _, err := os.Stat(dataDvdcssPath); err == nil {
			if err = utl.CopyFolder(dataDvdcssPath, roamingDvdcssPath); err != nil {
				Log.Warn().Err(err).Msgf("Cannot copy back %s", dataDvdcssPath)
			}
		}
		if _, err := os.Stat(roamingMlXspf); err == nil {
			if err = utl.CopyFile(roamingMlXspf, dataMlXspf); err != nil {
				Log.Warn().Err(err).Msgf("Cannot copy back %s", roamingMlXspf)
			}
		}
		if _, err := os.Stat(roamingVlcQtInterface); err == nil {
			if err = utl.CopyFile(roamingVlcQtInterface, dataVlcQtInterface); err != nil {
				Log.Warn().Err(err).Msgf("Cannot copy back %s", roamingVlcQtInterface)
			}
		}

		// Export registry key
		os.Remove(regFile)
		if err := registry.ExportKey(regKey); err != nil {
			Log.Warn().Err(err).Msg("Cannot export registry key")
		}

		// Remove tmp and roaming path
		os.RemoveAll(vlcTmpPath)
		os.RemoveAll(vlcRoamingPath)
	}()

	// Launch
	app.Launch(os.Args[1:])
}
