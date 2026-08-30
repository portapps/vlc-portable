//go:generate go install -v github.com/josephspurrier/goversioninfo/cmd/goversioninfo
package main

import (
	"os"
	"path/filepath"

	"github.com/portapps/portapps/v3"
	"github.com/portapps/portapps/v3/pkg/files"
	"github.com/portapps/portapps/v3/pkg/log"
	"github.com/portapps/portapps/v3/pkg/registry"
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
	if err := os.MkdirAll(app.DataPath, os.ModePerm); err != nil {
		log.Fatal().Err(err).Msg("Cannot create data directory.")
	}
	app.Process = filepath.Join(app.AppPath, "vlc.exe")
	app.Args = []string{
		"--vlm-conf=" + filepath.Join(app.DataPath, "vlcrc"),
		"--config=" + filepath.Join(app.DataPath, "vlcrc"),
		"--no-plugins-cache",
		"--no-qt-updates-notif",
	}

	// VLC paths
	vlcRoamingPath := filepath.Join(os.Getenv("APPDATA"), "vlc")
	vlcTmpPath := filepath.Join(app.AppPath, "tmp")
	if err := os.MkdirAll(vlcTmpPath, os.ModePerm); err != nil {
		log.Fatal().Err(err).Msg("Cannot create temporary directory.")
	}

	// Set env vars
	vlcPluginPath := filepath.Join(app.DataPath, "plugins")
	if err := os.MkdirAll(vlcPluginPath, os.ModePerm); err != nil {
		log.Fatal().Err(err).Msg("Cannot create plugins directory.")
	}
	os.Setenv("VLC_PLUGIN_PATH", vlcPluginPath)
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
	if err := os.MkdirAll(vlcRoamingPath, os.ModePerm); err != nil {
		log.Fatal().Err(err).Msg("Cannot create VLC roaming directory.")
	}
	if _, err := os.Stat(dataMlXspf); err == nil {
		if err := files.CopyFile(dataMlXspf, roamingMlXspf); err != nil {
			log.Error().Err(err).Msgf("Cannot copy %s", dataMlXspf)
		}
	}
	if _, err := os.Stat(dataVlcQtInterface); err == nil {
		if err := files.CopyFile(dataVlcQtInterface, roamingVlcQtInterface); err != nil {
			log.Error().Err(err).Msgf("Cannot copy %s", dataVlcQtInterface)
		}
	}

	// Handle reg key
	regPath := filepath.Join(app.RootPath, "reg")
	if err := os.MkdirAll(regPath, os.ModePerm); err != nil {
		log.Fatal().Err(err).Msg("Cannot create registry directory.")
	}
	regFile := filepath.Join(regPath, "VLC.reg")
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
			if err = files.CopyFolder(dataDvdcssPath, roamingDvdcssPath); err != nil {
				log.Warn().Err(err).Msgf("Cannot copy back %s", dataDvdcssPath)
			}
		}
		if _, err := os.Stat(roamingMlXspf); err == nil {
			if err = files.CopyFile(roamingMlXspf, dataMlXspf); err != nil {
				log.Warn().Err(err).Msgf("Cannot copy back %s", roamingMlXspf)
			}
		}
		if _, err := os.Stat(roamingVlcQtInterface); err == nil {
			if err = files.CopyFile(roamingVlcQtInterface, dataVlcQtInterface); err != nil {
				log.Warn().Err(err).Msgf("Cannot copy back %s", roamingVlcQtInterface)
			}
		}

		// Export reg key
		if err := regKey.Export(regFile); err != nil {
			log.Error().Err(err).Msg("Cannot export registry key")
		}

		// Cleanup
		if cfg.Cleanup {
			files.Cleanup(
				vlcRoamingPath,
				vlcTmpPath,
			)
			if err := regKey.Delete(true); err != nil {
				log.Error().Err(err).Msg("Cannot remove registry key")
			}
		}
	}()

	defer app.Close()
	app.Launch(os.Args[1:])
}
