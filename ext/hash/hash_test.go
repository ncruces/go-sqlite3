package hash

import (
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "golang.org/x/crypto/blake2b"
	_ "golang.org/x/crypto/blake2s"
	_ "golang.org/x/crypto/md4"
	_ "golang.org/x/crypto/ripemd160"
	_ "golang.org/x/crypto/sha3"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hash string
	}{
		{"md4(NULL)", ""},
		{"md4(X'')", "31D6CFE0D16AE931B73C59D7E0C089C0"},
		{"md4('The quick brown fox jumps over the lazy dog')", "1BEE69A46BA811185C194762ABAEAE90"},

		{"md5('')", "D41D8CD98F00B204E9800998ECF8427E"},
		{"sha1('')", "DA39A3EE5E6B4B0D3255BFEF95601890AFD80709"},
		{"ripemd160('')", "9C1185A5C5E9FC54612808977EE8F548B2258D31"},

		{"sha224('')", "D14A028C2A3A2BC9476102BB288234C415A2B01F828EA62AC5B3E42F"},
		{"sha256('')", "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"},
		{"sha256('', 224)", "D14A028C2A3A2BC9476102BB288234C415A2B01F828EA62AC5B3E42F"},
		{"sha384('')", "38B060A751AC96384CD9327EB1B1E36A21FDB71114BE07434C0CC7BF63F6E1DA274EDEBFE76F65FBD51AD2F14898B95B"},
		{"sha512('')", "CF83E1357EEFB8BDF1542850D66D8007D620E4050B5715DC83F4A921D36CE9CE47D0D13C5D85F2B0FF8318D2877EEC2F63B931BD47417A81A538327AF927DA3E"},
		{"sha512('', 224)", "6ED0DD02806FA89E25DE060C19D3AC86CABB87D6A0DDD05C333B84F4"},
		{"sha512('', 256)", "C672B8D1EF56ED28AB87C3622C5114069BDD3AD7B8F9737498D0C01ECEF0967A"},
		{"sha512('', 384)", "38B060A751AC96384CD9327EB1B1E36A21FDB71114BE07434C0CC7BF63F6E1DA274EDEBFE76F65FBD51AD2F14898B95B"},

		{"sha3('')", "A7FFC6F8BF1ED76651C14756A061D662F580FF4DE43B49FA82D80A4B80F8434A"},
		{"sha3('', 224)", "6B4E03423667DBB73B6E15454F0EB1ABD4597F9A1B078E3F5B5A6BC7"},
		{"sha3('', 384)", "0C63A75B845E4F7D01107D852E4C2485C51A50AAAA94FC61995E71BBEE983A2AC3713831264ADB47FB6BD1E058D5F004"},
		{"sha3('', 512)", "A69F73CCA23A9AC5C8B567DC185A756E97C982164FE25859E0D1DCC1475C80A615B2123AF1F5F94C11E3E9402C3AC558F500199D95B6D3E301758586281DCD26"},

		{"blake2s('')", "69217A3079908094E11121D042354A7C1F55B6482CA1A51E1B250DFD1ED0EEF9"},
		{"blake2b('')", "786A02F742015903C6C6FD852552D272912F4740E15847618A86E217F71F5419D25E1031AFEE585313896444934EB04B903A685B1448B755D56F701AFE9BE2CE"},
		{"blake2b('', 384)", "B32811423377F52D7862286EE1A72EE540524380FDA1724A6F25D7978C6FD3244A6CAF0498812673C5E05EF583825100"},
		{"blake2b('', 256)", "0E5751C026E543B2E8AB2EB06099DAA1D1E5DF47778F7787FAAB45CDF12FE3A8"},
	}

	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		Register(c)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hash string

			err = db.QueryRow(`SELECT hex(` + tt.name + `)`).Scan(&hash)
			if err != nil {
				t.Fatal(err)
			}

			if hash != tt.hash {
				t.Errorf("got %s, want %s", hash, tt.hash)
			}
		})
	}

	_, err = db.Exec(`SELECT sha256('', 255)`)
	if err == nil {
		t.Error("want error")
	}

	_, err = db.Exec(`SELECT sha512('', 255)`)
	if err == nil {
		t.Error("want error")
	}

	_, err = db.Exec(`SELECT sha3('', 255)`)
	if err == nil {
		t.Error("want error")
	}

	_, err = db.Exec(`SELECT blake2b('', 255)`)
	if err == nil {
		t.Error("want error")
	}
}
