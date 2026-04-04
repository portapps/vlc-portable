//go:generate go install -v github.com/josephspurrier/goversioninfo/cmd/goversioninfo
package main

import (
	"os"
	"path/filepath"

	"github.com/portapps/portapps/v3"
	"github.com/portapps/portapps/v3/pkg/log"
	"github.com/portapps/portapps/v3/pkg/registry"
	"github.com/portapps/portapps/v3/pkg/utl"
)

type config struct {
	Cleanup bool   `yaml:"cleanup" mapstructure:"cleanup"`
	Verbose string `yaml:"verbose" mapstructure:"verbose"`
}

var (
	app *portapps.App
	cfg *config
)

func init() {
	var err error

	// Default config
	cfg = &config{
		Cleanup: false,
		Verbose: "1",
	}

	// Init app
	if app, err = portapps.NewWithCfg("vlc-portable", "VLC", cfg); err != nil {
		log.Fatal().Err(err).Msg("Cannot initialize application. See log file for more info.")
	}
}

func main() {
	utl.CreateFolder(app.DataPath)
	app.Process = filepath.Join(app.AppPath, "vlc.exe")
	app.Args = []string{
		"--vlm-conf=" + filepath.Join(app.DataPath, "vlcrc"),
		"--config=" + filepath.Join(app.DataPath, "vlcrc"),
		"--no-plugins-cache",
		"--no-qt-updates-notif",
	}

	// VLC paths
	vlcRoamingPath := filepath.Join(os.Getenv("APPDATA"), "vlc")
	vlcTmpPath := utl.CreateFolder(app.AppPath, "tmp")

	// Set env vars
	os.Setenv("VLC_PLUGIN_PATH", utl.CreateFolder(app.DataPath, "plugins"))
	os.Setenv("VLC_VERBOSE", cfg.Verbose)
	os.Setenv("TEMP", vlcTmpPath)

	// VLC volatile files
	dataDvdcssPath := filepath.Join(app.DataPath, "dvdcss")
	dataMlXspf := filepath.Join(app.DataPath, "ml.xspf")
	dataVlcQtInterface := filepath.Join(app.DataPath, "vlc-qt-interface.ini")
	roamingDvdcssPath := filepath.Join(os.Getenv("APPDATA"), "dvdcss")
	roamingMlXspf := filepath.Join(vlcRoamingPath, "ml.xspf")
	roamingVlcQtInterface := filepath.Join(vlcRoamingPath, "vlc-qt-interface.ini")

	// Copy existing files from data to roaming folder for the current user
	utl.CreateFolder(vlcRoamingPath)
	if _, err := os.Stat(dataMlXspf); err == nil {
		if err := utl.CopyFile(dataMlXspf, roamingMlXspf); err != nil {
			log.Error().Err(err).Msgf("Cannot copy %s", dataMlXspf)
		}
	}
	if _, err := os.Stat(dataVlcQtInterface); err == nil {
		if err := utl.CopyFile(dataVlcQtInterface, roamingVlcQtInterface); err != nil {
			log.Error().Err(err).Msgf("Cannot copy %s", dataVlcQtInterface)
		}
	}

	// Handle reg key
	regFile := filepath.Join(utl.CreateFolder(app.RootPath, "reg"), "VLC.reg")
	regKey := registry.Key{
		Key:  `HKCU\Software\VideoLAN\VLC`,
		Arch: "32",
	}
	if err := regKey.Import(regFile); err != nil {
		log.Warn().Err(err).Msg("Cannot import registry key")
	}

	// On exit
	defer func() {
		// Copy back to data
		if _, err := os.Stat(dataDvdcssPath); err == nil {
			if err = utl.CopyFolder(dataDvdcssPath, roamingDvdcssPath); err != nil {
				log.Warn().Err(err).Msgf("Cannot copy back %s", dataDvdcssPath)
			}
		}
		if _, err := os.Stat(roamingMlXspf); err == nil {
			if err = utl.CopyFile(roamingMlXspf, dataMlXspf); err != nil {
				log.Warn().Err(err).Msgf("Cannot copy back %s", roamingMlXspf)
			}
		}
		if _, err := os.Stat(roamingVlcQtInterface); err == nil {
			if err = utl.CopyFile(roamingVlcQtInterface, dataVlcQtInterface); err != nil {
				log.Warn().Err(err).Msgf("Cannot copy back %s", roamingVlcQtInterface)
			}
		}

		// Export reg key
		if err := regKey.Export(regFile); err != nil {
			log.Error().Err(err).Msg("Cannot export registry key")
		}

		// Cleanup
		if cfg.Cleanup {
			utl.Cleanup([]string{
				vlcRoamingPath,
				vlcTmpPath,
			})
			if err := regKey.Delete(true); err != nil {
				log.Error().Err(err).Msg("Cannot remove registry key")
			}
		}
	}()

	defer app.Close()
	app.Launch(os.Args[1:])
}
