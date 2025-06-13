package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Configuration struct {
	Name       string   // Short name used for binary names, mention on command line
	Root       string   // Specific Go root to use for this trial
	PgoGen     bool     // Generate profiles for each configuration (path: temporary directory / config name / bench name / profiles)
	PgoUse     string   // Name of configuration that generated profiles (PGO disabled if empty)
	BuildFlags []string // BuildFlags supplied to 'go test -c' for building (e.g., "-p 1")
	GcFlags    string   // GcFlags supplied to 'go test -c' for building
	LdFlags    string   // LdFlags supplied to 'go test -c' for building
	GcEnv      []string // Environment variables supplied to 'go test -c' for building
	RunFlags   []string // Extra flags passed to the test binary
	RunEnv     []string // Extra environment variables passed to the test binary
	RunWrapper []string // (Outermost) Command and args to precede whatever the operation is; may fail in the sandbox.
	Disabled   bool     // True if this configuration is temporarily disabled
}

type ConfList struct {
	Configurations []Configuration
}

type Benchmark struct {
	Name       string   // Short name for benchmark/test
	Benchmarks string   // Benchmarks to run (regex for -test.bench= )
	Dir        string   // Path to directory where benchmarks are stored
	RunFlags   []string // Extra flags passed to the test binary
	Disabled   bool     // True if this benchmark is temporarily disabled
}

type BenchList struct {
	Benchmarks []Benchmark
}

// Creates directory and returns its absolute path.
func initDir(dir string) string {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal(err)
	}
	return abs
}

func buildBenchmarks(c Configuration, bench []Benchmark, tmpDir string, verbose bool) {
	benchTmpDir := initDir(filepath.Join(tmpDir, c.Name))
	root := os.ExpandEnv(c.Root)
	root, err := filepath.Abs(root)
	if err != nil {
		log.Fatal(err)
	}
	profDir := ""
	if c.PgoUse != "" {
		profDir = initDir(filepath.Join(tmpDir, c.PgoUse, "profiles"))
	}
	fmt.Printf("Building benchmarks for configuration '%s'...\n", c.Name)
	gocmd := filepath.Join(root, "bin", "go")
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	for _, b := range bench {
		if b.Disabled {
			continue
		}
		err = os.Chdir(pwd)
		if err != nil {
			log.Fatal(err)
		}
		bdir := os.ExpandEnv(b.Dir)
		bdir, err := filepath.Abs(bdir)
		if err != nil {
			log.Fatal(err)
		}
		if verbose {
			fmt.Printf("( cd %v )\n", bdir)
		}
		err = os.Chdir(bdir)
		if err != nil {
			log.Fatal(err)
		}
		cmd := exec.Command(gocmd, "test", "-c", "-a")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, c.GcEnv...)
		goroot := "GOROOT=" + root
		cmd.Env = append(cmd.Env, goroot)
		cmd.Args = append(cmd.Args, "-gcflags=all="+c.GcFlags)
		cmd.Args = append(cmd.Args, "-ldflags=all="+c.LdFlags)
		cmd.Args = append(cmd.Args, c.BuildFlags...)
		if c.PgoUse != "" {
			benchProf := filepath.Join(profDir, b.Name+".pprof")
			cmd.Args = append(cmd.Args, "-pgo="+benchProf)
		}
		cmd.Args = append(cmd.Args, "-o", filepath.Join(benchTmpDir, b.Name))
		if verbose {
			cmdstr := goroot + " " + cmd.String()
			for _, env := range c.GcEnv {
				cmdstr = env + " " + cmdstr
			}
			fmt.Printf("( %s )\n", cmdstr)
		}
		fmt.Println()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func runBenchmarks(c Configuration, bench []Benchmark, tmpDir string, count int, verbose bool) {
	benchTmpDir := initDir(filepath.Join(tmpDir, c.Name))
	profDir := ""
	if c.PgoGen {
		profDir = initDir(filepath.Join(benchTmpDir, "profiles"))
	}
	fmt.Printf("Running benchmarks for configuration '%s'...\n", c.Name)
	for _, b := range bench {
		if b.Disabled {
			continue
		}
		profTmpDir := ""
		if c.PgoGen {
			profTmpDir = initDir(filepath.Join(profDir, "_"+b.Name))
			err := os.RemoveAll(profTmpDir)
			if err != nil {
				log.Fatal(err)
			}
			profTmpDir = initDir(profTmpDir)
		}
		for i := 0; i < count; i++ {
			if verbose {
				fmt.Printf("( cd %v )\n", benchTmpDir)
			}
			err := os.Chdir(benchTmpDir)
			if err != nil {
				log.Fatal(err)
			}
			bin, err := filepath.Abs(filepath.Join(benchTmpDir, b.Name))
			if err != nil {
				log.Fatal(err)
			}
			var cmd *exec.Cmd
			if len(c.RunWrapper) > 0 {
				for i, wrap := range c.RunWrapper {
					if i == 0 {
						cmd = exec.Command(wrap)
						continue
					}
					cmd.Args = append(cmd.Args, wrap)
				}
				cmd.Args = append(cmd.Args, bin)
			} else {
				cmd = exec.Command(bin)
			}

			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, c.RunEnv...)

			cmd.Args = append(cmd.Args, "-test.bench="+b.Benchmarks)
			cmd.Args = append(cmd.Args, c.RunFlags...)
			cmd.Args = append(cmd.Args, b.RunFlags...)
			if c.PgoGen {
				prof := fmt.Sprintf("%s_%d.pprof", b.Name, i)
				benchProf := filepath.Join(profTmpDir, prof)
				cmd.Args = append(cmd.Args, "-test.cpuprofile="+benchProf)
			}
			if verbose {
				cmdstr := cmd.String()
				for _, env := range c.RunEnv {
					cmdstr = env + " " + cmdstr
				}
				fmt.Printf("( %s )\n", cmdstr)
			}
			fmt.Println()
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Printf("shortname: %s\ntoolchain: %s\n", b.Name, c.Name)
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}
		if c.PgoGen && count > 0 {
			root := os.ExpandEnv(c.Root)
			root, err := filepath.Abs(root)
			if err != nil {
				log.Fatal(err)
			}
			gocmd := filepath.Join(root, "bin", "go")
			profs := make([]string, 0)
			for i := 0; i < count; i++ {
				prof := fmt.Sprintf("%s_%d.pprof", b.Name, i)
				prof = filepath.Join(profTmpDir, prof)
				profs = append(profs, prof)
			}
			cmd := exec.Command(gocmd, "tool", "pprof", "-proto")
			cmd.Args = append(cmd.Args, profs...)
			if verbose {
				cmdstr := cmd.String()
				for _, env := range c.RunEnv {
					cmdstr = env + " " + cmdstr
				}
				fmt.Printf("( %s )\n", cmdstr)
			}
			fmt.Println()

			merged := fmt.Sprintf("%s.pprof", b.Name)
			merged = filepath.Join(profDir, merged)
			out, err := os.Create(merged)
			if err != nil {
				log.Fatal(err)
			}
			defer out.Close()
			cmd.Stdout = out
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
			err = os.RemoveAll(profTmpDir)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {
	conf := flag.String("C", "", "configurations file")
	confNames := flag.String("c", "", "use configurations from comma-separated list (even if normally \"disabled\")")
	verbose := flag.Bool("v", false, "print commands as they are run")
	count := flag.Int("N", 1, "benchmark/test repeat count")
	bench := flag.String("B", "", "benchmarks file")
	benchNames := flag.String("b", "", "run benchmarks in comma-separated list (even if normally \"disabled\" )")
	tmpDir := flag.String("T", "tmp", "path to temporary directory")
	flag.Parse()

	if *conf == "" {
		fmt.Fprintln(os.Stderr, "Configurations file expected but not presented.")
		os.Exit(1)
	}
	if *bench == "" {
		fmt.Fprintln(os.Stderr, "Benchmarks file expected but not presented.")
		os.Exit(1)
	}

	var confList ConfList
	_, err := toml.DecodeFile(*conf, &confList)
	if err != nil {
		log.Fatal(err)
	}

	var benchList BenchList
	_, err = toml.DecodeFile(*bench, &benchList)
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range confList.Configurations {
		if c.Name == "" {
			fmt.Fprintf(os.Stderr, "ERROR: each configuration must have a name!")
			os.Exit(1)
		}
	}

	for _, b := range benchList.Benchmarks {
		if b.Name == "" {
			fmt.Fprintf(os.Stderr, "ERROR: each benchmark must have a name!")
			os.Exit(1)
		}
	}

	if len(*confNames) != 0 {
		names := strings.Split(*confNames, ",")
		nameMap := make(map[string]struct{})
		for _, name := range names {
			if len(name) > 0 {
				nameMap[name] = struct{}{}
			}
		}
		for i := 0; i < len(confList.Configurations); i++ {
			c := &confList.Configurations[i]
			_, ok := nameMap[c.Name]
			c.Disabled = !ok
		}
	}

	if len(*benchNames) != 0 {
		names := strings.Split(*benchNames, ",")
		nameMap := make(map[string]struct{})
		for _, name := range names {
			if len(name) > 0 {
				nameMap[name] = struct{}{}
			}
		}
		for i := 0; i < len(benchList.Benchmarks); i++ {
			b := &benchList.Benchmarks[i]
			_, ok := nameMap[b.Name]
			b.Disabled = !ok
		}
	}

	for i := range benchList.Benchmarks {
		b := &benchList.Benchmarks[i]
		if b.Disabled {
			continue
		}
		if b.Dir == "" {
			fmt.Printf("WARNING: 'Dir' is not set for benchmark '%s'. Use current directory.\n", b.Name)
			b.Dir = "."
		}
		if b.Benchmarks == "" {
			b.Benchmarks = "Benchmark"
		}
	}

	tmp := initDir(*tmpDir)
	for _, c := range confList.Configurations {
		if c.Disabled {
			continue
		}
		if c.PgoGen && c.PgoUse != "" {
			fmt.Fprintf(os.Stderr, "WARNING: PgoGen is set for configuration '%s', ignore PgoUse.\n", c.Name)
			c.PgoUse = ""
		}
		buildBenchmarks(c, benchList.Benchmarks, tmp, *verbose)
		runBenchmarks(c, benchList.Benchmarks, tmp, *count, *verbose)
	}
}
