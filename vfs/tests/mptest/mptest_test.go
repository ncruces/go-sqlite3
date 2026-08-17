package mptest

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	_ "unsafe"

	_ "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/internal/testenv"
	"github.com/ncruces/go-sqlite3/vfs"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/ncruces/go-sqlite3/vfs/mvcc"
	_ "github.com/ncruces/go-sqlite3/vfs/mvcc"
	_ "github.com/ncruces/go-sqlite3/vfs/xts"
)

const ptrlen = sqlite3_wrap.PtrLen

type ptr_t = sqlite3_wrap.Ptr_t

//go:linkname createWrapper github.com/ncruces/go-sqlite3.createWrapper
func createWrapper(ctx context.Context) (*sqlite3_wrap.Wrapper, error)

//go:embed testdata/*
var scripts embed.FS

func init() {
	testenv.Exit = exit
	testenv.System = system
	testenv.FS, _ = fs.Sub(scripts, "testdata")
}

func runTest(t *testing.T, args ...string) {
	testenv.TB = t
	wrp, err := createWrapper(testcfg.Context(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	argv := wrp.New(int64(ptrlen * len(args)))
	for i, a := range args {
		wrp.Write32(argv+ptr_t(i)*ptrlen, uint32(wrp.NewString(a)))
	}

	if c := wrp.Xmain_mptest(int32(len(args)), int32(argv)); c != 0 {
		t.Error("exit error: ", c)
	}
}

func Test_mptest(t *testing.T) {
	scripts := []struct {
		script      string
		slow        bool
		crashes     bool
		changesJrnl bool
		changesPgsz bool
	}{
		{script: "multiwrite01.test"},
		{script: "crash01.test", crashes: true},
		{script: "config01.test", changesJrnl: true},
		{script: "config02.test", slow: true, crashes: true, changesPgsz: true},
	}

	envs := []struct {
		name    string
		vfs     string
		isMemdb bool
		isMVCC  bool
		isWAL   bool
		needKey bool
	}{
		{name: ""},
		{name: "_wal", isWAL: true},
		// Encryption.
		{name: "_xts", vfs: "xts", needKey: true},
		{name: "_adiantum", vfs: "adiantum", needKey: true},
		{name: "_xts_wal", vfs: "xts", isWAL: true, needKey: true},
		{name: "_adiantum_wal", vfs: "adiantum", isWAL: true, needKey: true},
		// Memory.
		{name: "_memory", vfs: "memdb", isMemdb: true},
		{name: "_mvcc", vfs: "mvcc", isMVCC: true},
	}

	for _, script := range scripts {
		for _, env := range envs {
			if env.isMemdb && script.crashes {
				continue
			}
			if env.isMVCC && script.changesPgsz {
				continue
			}
			if env.isWAL && (script.changesJrnl || script.changesPgsz) {
				continue
			}

			name := strings.TrimSuffix(script.script, ".test") + env.name
			t.Run(name, func(t *testing.T) {
				if !vfs.SupportsFileLocking && !(env.isMemdb || env.isMVCC) {
					t.Skip("skipping without file locking")
				}
				if !vfs.SupportsSharedMemory && env.isWAL {
					t.Skip("skipping without shared memory")
				}
				if os.Getenv("CI") != "" && script.slow {
					t.Skip("skipping in CI")
				}
				if testing.Short() && (script.slow || env.needKey) {
					t.Skip("skipping in short mode")
				}

				var db string
				switch {
				case env.isMemdb:
					db = memdb.TestDB(t)
				case env.isMVCC:
					db = mvcc.TestDB(t, mvcc.Snapshot{})
				default:
					db = filepath.Join(t.TempDir(), "test.db")
					if env.needKey {
						db = "file:" + db +
							"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
					}
				}

				args := []string{"mptest", db, script.script}
				if env.vfs != "" {
					args = append(args, "--vfs", env.vfs)
				}
				if env.isWAL {
					args = append(args, "--journalmode", "wal")
				}

				runTest(t, args...)
			})
		}
	}
}

func system(wrp *sqlite3_wrap.Wrapper, ptr int32) int32 {
	if ptr == 0 {
		return 0
	}

	s := wrp.ReadString(ptr_t(ptr), 1e6)

	args := strings.Split(s, " ")
	for i := range args {
		args[i] = strings.Trim(args[i], `"`)
	}
	if args[0] != "mptest" || args[len(args)-1] != "&" {
		return -1
	}
	args = args[:len(args)-1]

	go func() {
		wrp, err := createWrapper(testcfg.Context(testenv.TB))
		if err != nil {
			panic(err)
		}
		defer wrp.Close()

		argv := wrp.New(int64(ptrlen * len(args)))
		for i, a := range args {
			wrp.Write32(argv+ptr_t(i)*ptrlen, uint32(wrp.NewString(a)))
		}

		defer func() { recover() }()
		wrp.Xmain_mptest(int32(len(args)), int32(argv))
	}()
	return 0
}

func exit(c int32) {
	if c != 0 {
		panic(fmt.Sprint("exit error: ", c))
	}
	runtime.Goexit()
}
