package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"

	"everyflac-tui/pkg/config"
	"everyflac-tui/pkg/downloader"
	"everyflac-tui/pkg/models"
	"everyflac-tui/pkg/spotify"
	"everyflac-tui/pkg/tui"
	"everyflac-tui/pkg/uninstaller"
	"everyflac-tui/pkg/updater"
	"everyflac-tui/pkg/ytmusic"
	tea "github.com/charmbracelet/bubbletea"
)

//go:embed VERSION
var versionFile string

// AppVersion is the single source of truth, read from VERSION at build time.
var AppVersion = strings.TrimSpace(versionFile)

func main() {
	tui.Version = AppVersion

	fs := flag.NewFlagSet("everyflac", flag.ExitOnError)

	outputFlag := fs.String("output", "", "Destination directory for downloaded FLAC audio (default: ~/Music)")
	fs.StringVar(outputFlag, "o", "", "Destination directory (shorthand)")

	urlFlag := fs.String("url", "", "Spotify or YouTube Music URL to download directly without opening TUI")
	fs.StringVar(urlFlag, "u", "", "URL (shorthand)")

	searchFlag := fs.String("search", "", "Search query to find and download top matching song directly")
	fs.StringVar(searchFlag, "s", "", "Search query (shorthand)")

	modeFlag := fs.String("mode", "", "Source engine mode: auto, spotify, youtube")
	fs.StringVar(modeFlag, "m", "", "Source engine mode (shorthand)")

	versionFlag := fs.Bool("version", false, "Print application version")
	fs.BoolVar(versionFlag, "v", false, "Print application version (shorthand)")

	checkUpdateFlag := fs.Bool("check-update", false, "Check for a newer release and exit")

	removeFlag := fs.Bool("remove", false, "Uninstall everyFLAC and remove binaries and shell configurations")
	fs.BoolVar(removeFlag, "uninstall", false, "Uninstall everyFLAC (alias for --remove)")

	yesFlag := fs.Bool("yes", false, "Confirm uninstallation without interactive prompt")
	fs.BoolVar(yesFlag, "y", false, "Confirm uninstallation without interactive prompt (shorthand)")

	fs.Usage = func() {
		fmt.Println(tui.GetASCIIBanner(0, false, false))
		fmt.Printf("\neveryFLAC v%s — all-in-one lossless FLAC downloader\n\n", AppVersion)
		fmt.Println("Usage:")
		fmt.Println("  everyflac                           Launch interactive TUI")
		fmt.Println("  everyflac <url_or_search>           Download directly from command line")
		fmt.Println("  everyflac --url <url>               Download Spotify or YouTube Music link directly")
		fmt.Println("  everyflac --search <query>          Search and download top match directly")
		fmt.Println("  everyflac --output <dir>            Set destination folder (default: ~/Music)")
		fmt.Println("  everyflac --mode <auto|spotify|yt>  Force source engine mode")
		fmt.Println("  everyflac --check-update            Check for a newer release and exit")
		fmt.Println("  everyflac --remove                  Uninstall and remove everyFLAC completely")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}

	reordered := rearrangeArgs(os.Args[1:])
	_ = fs.Parse(reordered)

	if *versionFlag {
		fmt.Printf("everyFLAC v%s\n", AppVersion)
		return
	}

	if *checkUpdateFlag {
		runUpdateCheck()
		return
	}

	if *removeFlag {
		if err := uninstaller.Run(*yesFlag); err != nil {
			fmt.Printf("Error during removal: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if *outputFlag != "" {
		cfg.OutputDir = config.ExpandPath(*outputFlag)
	} else {
		cfg.OutputDir = config.ExpandPath(cfg.OutputDir)
	}

	if *modeFlag != "" {
		m := strings.ToLower(strings.TrimSpace(*modeFlag))
		if m == "yt" || m == "ytmusic" || m == "youtube" {
			cfg.SourceMode = "youtube"
		} else if m == "sp" || m == "spotify" {
			cfg.SourceMode = "spotify"
		} else {
			cfg.SourceMode = "auto"
		}
	}

	cliInput := ""
	if *urlFlag != "" {
		cliInput = *urlFlag
	} else if *searchFlag != "" {
		cliInput = *searchFlag
	} else if fs.NArg() > 0 {
		cliInput = strings.Join(fs.Args(), " ")
	}

	if cliInput != "" {
		runHeadlessCLI(cliInput, cfg)
		return
	}

	model := tui.NewModel(cfg)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	// Ask the terminal to report Ctrl+1/Ctrl+2 distinctly. Level 1 only
	// affects keys without a legacy control code, so Ctrl+A/O/H keep working.
	// Terminals that do not implement modifyOtherKeys simply ignore this.
	fmt.Print("\x1b[>4;1m")

	_, err = program.Run()
	fmt.Print("\x1b[>4;0m")
	if err != nil {
		fmt.Printf("Error running everyFLAC by pxradise: %v\n", err)
		os.Exit(1)
	}
}

// runUpdateCheck performs an explicit, cache-bypassing update check and prints
// the result. It is what `everyflac --check-update` runs.
func runUpdateCheck() {
	res, err := updater.CheckFresh(AppVersion)
	if err != nil {
		fmt.Printf("everyFLAC v%s: update check failed: %v\n", AppVersion, err)
		os.Exit(1)
	}
	if res.Available {
		fmt.Printf("everyFLAC v%s: update available: v%s\n", res.Current, res.Latest)
		fmt.Printf("https://github.com/%s/releases/latest\n", updater.Repo)
		return
	}
	fmt.Printf("everyFLAC v%s: up to date\n", res.Current)
}

func rearrangeArgs(args []string) []string {
	var flags []string
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)

			if !strings.Contains(arg, "=") && (arg == "-o" || arg == "--output" || arg == "-u" || arg == "--url" || arg == "-s" || arg == "--search" || arg == "-m" || arg == "--mode") {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...)
}

func runHeadlessCLI(input string, cfg *config.AppConfig) {
	fmt.Println(tui.GetASCIIBanner(0, cfg.SourceMode == "youtube", false))
	fmt.Printf("\neveryFLAC CLI v%s\n", AppVersion)
	fmt.Printf("engine: %s\n", strings.ToLower(cfg.SourceMode))
	fmt.Printf("output: %s\n\n", cfg.OutputDir)

	input = strings.TrimSpace(input)
	var batch *models.DownloadBatch
	var err error

	mode := cfg.SourceMode

	if mode == "youtube" && spotify.IsSpotifyURL(input) {

		link, pErr := spotify.ParseLink(input)
		if pErr == nil {
			spClient := spotify.NewClient()
			batch, err = spClient.FetchBatch(link)
			if err == nil && batch != nil {
				batch.Platform = models.PlatformYouTubeMusic
				for _, t := range batch.Tracks {
					t.Platform = models.PlatformYouTubeMusic
					t.SourceURL = fmt.Sprintf("ytsearch1:%s - %s", t.Artist, t.Title)
				}
			}
		}
	} else if (mode == "auto" || mode == "spotify") && spotify.IsSpotifyURL(input) {
		link, pErr := spotify.ParseLink(input)
		if pErr != nil {
			fmt.Printf("error: %v\n", pErr)
			os.Exit(1)
		}
		fmt.Println("resolving spotify metadata...")
		spClient := spotify.NewClient()
		batch, err = spClient.FetchBatch(link)
	} else {
		fmt.Printf("resolving metadata for %q...\n", input)
		ytClient := ytmusic.NewClient()
		batch, err = ytClient.ResolveInput(input)
		if err == nil && batch != nil && mode == "spotify" {
			batch.Platform = models.PlatformSpotify
			for _, t := range batch.Tracks {
				t.Platform = models.PlatformSpotify
			}
		}
	}

	if err != nil {
		fmt.Printf("error resolving metadata: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("loaded: %s by %s (%d tracks, %s)\n\n",
		batch.Name, batch.Subtitle, len(batch.Tracks), batch.Platform)

	dl := downloader.NewDownloader(cfg)
	for i, track := range batch.Tracks {
		fmt.Printf("[%d/%d] %s - %s\n", i+1, len(batch.Tracks), track.Artist, track.Title)

		prog, dlErr := dl.DownloadTrack(track, batch, func(p *models.TrackProgress) {
			if p.Status == models.StatusDownloading && p.ProgressPercent > 0 {
				speed := ""
				if p.SpeedBytesPerSec > 0 {
					speed = fmt.Sprintf(" (%.1f MB/s)", float64(p.SpeedBytesPerSec)/(1024*1024))
				}
				fmt.Printf("\r  downloading %4.1f%%%s", p.ProgressPercent, speed)
			}
		})

		fmt.Println()
		if dlErr != nil {
			fmt.Printf("  failed: %v\n\n", dlErr)
		} else {
			fmt.Printf("  saved: %s (%s, %s)\n\n", prog.FilePath, prog.QualityLabel, prog.SourceUsed)
		}
	}

	fmt.Println("done.")
}
