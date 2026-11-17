package approx

import (
	"fmt"
	"math"
	"strconv"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func sketchInfo(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		data := arg[0].RawBlob()
		if len(data) == 0 {
			ctx.ResultError(errutil.ErrorString("approx: insufficient bytes to deserialize sketch"))
			return
		}
		switch data[0] {
		case 0x11, 0x12, 0x13, 0x14:
			hllInfo(ctx, data)
		case reqHeader:
			reqInfo(ctx, data)
		case topHeader:
			topInfo(ctx, data)
		default:
			ctx.ResultError(errutil.ErrorString("approx: unknown sketch format"))
		}
	default:
		ctx.ResultError(fmt.Errorf("approx: invalid type: %v", t))
	}
}

func hllInfo(ctx sqlite3.Context, data []byte) {
	sketch, err := hllFromBytes(data)
	if err != nil {
		ctx.ResultError(fmt.Errorf("approx: %w", err))
		return
	}

	count := sketch.Cardinality()
	var stderr, rse float64
	if typ := data[0] & 0xf; typ >= 3 && count > 0 {
		m := float64(uint64(1) << sketch.Settings().Log2m)
		if n := float64(count); n < 2.5*m {
			// Small-range LinearCounting regime (Whang et al., 1990).
			t := n / m
			stderr = math.Sqrt(m * (math.Expm1(t) - t))
			rse = stderr / n
		} else {
			// HyperLogLog regime (Flajolet et al., 2007, Theorem 1).
			const beta = 1.0389617614136893 // sqrt(3*ln(2) - 1)
			rse = beta / math.Sqrt(m)
			stderr = rse * n
		}
	}

	var buf [128]byte
	out := append(buf[:0], `{"type":"count_distinct","count":`...)
	out = strconv.AppendUint(out, count, 10)
	out = append(out, `,"stderr":`...)
	out = util.AppendFloat(out, stderr, 64)
	out = append(out, `,"rse":`...)
	out = util.AppendFloat(out, rse, 64)
	out = append(out, '}')
	ctx.ResultRawText(out)
	ctx.ResultSubtype('J')
}

func reqInfo(ctx sqlite3.Context, data []byte) {
	sketch, err := reqFromBytes(data)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	var buf [192]byte
	out := append(buf[:0], `{"type":"percentile","count":`...)
	if len(sketch.levels) == 0 {
		out = append(out, `0,"min":null,"max":null,"avg":null,"stddev":null,"rank_rse":0.0}`...)
		ctx.ResultRawText(out)
		ctx.ResultSubtype('J')
		return
	}

	var (
		count  uint64
		mean   float64
		m2     float64
		stddev = math.NaN()
		min32  = sketch.levels[0].buf[0]
		max32  = min32
	)
	for h := range sketch.levels {
		w := uint64(1) << h
		wf := float64(w)
		for _, x := range sketch.levels[h].buf {
			count += w
			min32 = min(min32, x)
			max32 = max(max32, x)
			fx := float64(x)
			d1 := fx - mean
			mean += (wf / float64(count)) * d1
			m2 += wf * d1 * (fx - mean)
		}
	}

	if math.IsInf(float64(min32), 0) || math.IsInf(float64(max32), 0) {
		mean = float64(min32) + float64(max32)
	} else if count >= 2 {
		stddev = math.Sqrt(max(m2, 0) / float64(count-1))
	}

	var rse float64
	if len(sketch.levels) > 1 {
		// Empirical 1-sigma relative rank error factor in HRA mode
		// with initial s=3, k=40 (Cormode et al., 2021):
		// sigma_rank(p) ≈ rse * (1 - p).
		rse = 0.0032659863237109041 // √(0.0512/s) / k
	}

	out = strconv.AppendUint(out, count, 10)
	out = append(out, `,"min":`...)
	out = util.AppendFloat(out, float64(min32), 32)
	out = append(out, `,"max":`...)
	out = util.AppendFloat(out, float64(max32), 32)
	out = append(out, `,"avg":`...)
	out = util.AppendFloat(out, float64(float32(mean)), 32)
	out = append(out, `,"stddev":`...)
	out = util.AppendFloat(out, stddev, 64)
	out = append(out, `,"rank_rse":`...)
	out = util.AppendFloat(out, rse, 64)
	out = append(out, '}')
	ctx.ResultRawText(out)
	ctx.ResultSubtype('J')
}

func topInfo(ctx sqlite3.Context, data []byte) {
	sketch, err := topFromBytes(data)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	var buf [128]byte
	out := append(buf[:0], `{"type":"top_k","k":`...)
	if sketch.k == 0 {
		out = append(out, "null"...)
	} else {
		out = strconv.AppendInt(out, int64(sketch.k), 10)
	}
	out = append(out, `,"total":`...)
	out = util.AppendNumber(out, sketch.total, 64)
	out = append(out, `,"max_error":`...)
	out = util.AppendNumber(out, sketch.offset, 64)
	out = append(out, '}')
	ctx.ResultRawText(out)
	ctx.ResultSubtype('J')
}
