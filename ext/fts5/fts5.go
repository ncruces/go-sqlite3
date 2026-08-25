// Package fts5 provides the fts5 extension.
//
// https://sqlite.org/fts5.html
package fts5

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3-wasm/v4/fts5"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

// Register registers the fts5 extension.
func Register(db *sqlite3.Conn) error {
	return RegisterCustom(db, nil)
}

func RegisterCustom(db *sqlite3.Conn, init func(*Api) error) error {
	var api Api
	err := sqlite3.ExtensionInit(db, func(e *sqlite3.ExtEnv) *fts5.Module {
		api.env = env{ExtEnv: e}
		api.env.Module = fts5.New(&api.env)
		return api.env.Module
	}, fts5.DylinkInfo)
	if err == nil && init != nil {
		return init(&api)
	}
	return err
}

type Api struct{ env env }

func (a *Api) CreateTokenizer(name string, fn TokenizerConstructor) error {
	ptr := a.env.NewString(name)
	defer a.env.Free(ptr)

	var handle ptr_t
	if fn != nil {
		handle = a.env.AddHandle(fn)
	}

	rc := a.env.Xfts5_xCreateTokenizer_v2(int32(ptr), int32(handle))
	return sql3util.CodeToError(rc)
}

type Tokenizer interface {
	Tokenize(flags TokenizeFlag, text, locale string,
		xToken func(tflags TokenFlag, token string, start, end int32) error) error
}

type TokenizerConstructor func(arg string) (Tokenizer, error)

type env struct {
	*sqlite3.ExtEnv
	*fts5.Module
}

func (e *env) Xgo_fts5_create(pApp, azArg, nArg, pOut int32) int32 {
	fn := e.GetHandle(ptr_t(pApp)).(TokenizerConstructor)
	t, err := fn(string(e.Bytes(ptr_t(azArg), int64(nArg))))

	var handle ptr_t
	if t != nil {
		handle = e.AddHandle(t)
	}
	e.Write32(ptr_t(pOut), uint32(handle))
	return sql3util.ErrorToCode(err)
}

func (e *env) Xgo_fts5_tokenize(pTok, pCtx, flags, pText, nText, pLocale, nLocale, xToken int32) int32 {
	var locale string
	tok := e.GetHandle(ptr_t(pTok)).(Tokenizer)
	text := string(e.Bytes(ptr_t(pText), int64(nText)))
	if pLocale != 0 {
		locale = string(e.Bytes(ptr_t(pLocale), int64(nLocale)))
	}

	err := tok.Tokenize(TokenizeFlag(flags), text, locale, func(tflags TokenFlag, token string, start, end int32) error {
		var ptr ptr_t
		var siz int32

		if token == "" {
			ptr = ptr_t(pText + start)
			siz = end - start
		} else {
			ptr := e.NewString(token)
			siz = int32(len(token))
			defer e.Free(ptr)
		}

		rc := e.Xfts5_xToken(xToken, pCtx, int32(tflags), int32(ptr), siz, start, end)
		return sql3util.CodeToError(rc)
	})

	return sql3util.ErrorToCode(err)
}

type ptr_t = sqlite3_wrap.Ptr_t
