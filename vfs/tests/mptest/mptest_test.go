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

func Test_config01(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := filepath.Join(t.TempDir(), "test.db")
	runTest(t, "mptest", name, "config01.test")
}

func Test_config02(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := filepath.Join(t.TempDir(), "test.db")
	runTest(t, "mptest", name, "config02.test")
}

func Test_crash01(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := filepath.Join(t.TempDir(), "test.db")
	runTest(t, "mptest", name, "crash01.test")
}

func Test_multiwrite01(t *testing.T) {
	if testing.Short() && os.Getenv("CI") != "" {
		t.Skip("skipping in slow CI")
	}
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := filepath.Join(t.TempDir(), "test.db")
	runTest(t, "mptest", name, "multiwrite01.test")
}

func Test_config01_memory(t *testing.T) {
	memdb.Create("test.db", nil)
	runTest(t, "mptest", "/test.db", "config01.test",
		"--vfs", "memdb")
}

func Test_multiwrite01_memory(t *testing.T) {
	if testing.Short() && os.Getenv("CI") != "" {
		t.Skip("skipping in slow CI")
	}

	memdb.Create("test.db", nil)
	runTest(t, "mptest", "/test.db", "multiwrite01.test",
		"--vfs", "memdb")
}

func Test_config01_mvcc(t *testing.T) {
	mvcc.Create("test.db", mvcc.Snapshot{})
	runTest(t, "mptest", "/test.db", "config01.test",
		"--vfs", "mvcc")
}

func Test_crash01_mvcc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	mvcc.Create("test.db", mvcc.Snapshot{})
	runTest(t, "mptest", "/test.db", "crash01.test",
		"--vfs", "mvcc")
}

func Test_multiwrite01_mvcc(t *testing.T) {
	if testing.Short() && os.Getenv("CI") != "" {
		t.Skip("skipping in slow CI")
	}

	mvcc.Create("test.db", mvcc.Snapshot{})
	runTest(t, "mptest", "/test.db", "multiwrite01.test",
		"--vfs", "mvcc")
}

func Test_crash01_wal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	name := filepath.Join(t.TempDir(), "test.db")
	runTest(t, "mptest", name, "crash01.test",
		"--journalmode", "wal")
}

func Test_multiwrite01_wal(t *testing.T) {
	if testing.Short() && os.Getenv("CI") != "" {
		t.Skip("skipping in slow CI")
	}
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	name := filepath.Join(t.TempDir(), "test.db")
	runTest(t, "mptest", name, "multiwrite01.test",
		"--journalmode", "wal")
}

func Test_crash01_adiantum(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := "file:" + filepath.Join(t.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	runTest(t, "mptest", name, "crash01.test",
		"--vfs", "adiantum")
}

func Test_crash01_adiantum_wal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	name := "file:" + filepath.Join(t.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	runTest(t, "mptest", name, "crash01.test",
		"--vfs", "adiantum", "--journalmode", "wal")
}

func Test_crash01_xts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := "file:" + filepath.Join(t.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	runTest(t, "mptest", name, "crash01.test",
		"--vfs", "xts")
}

func Test_crash01_xts_wal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	name := "file:" + filepath.Join(t.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	runTest(t, "mptest", name, "crash01.test",
		"--vfs", "xts", "--journalmode", "wal")
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
