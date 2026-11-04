package restapi

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// _uploadsMaxAge is the browser cache lifetime of uploaded files, in seconds. Stored names never
// change, so a day of caching is safe.
const _uploadsMaxAge = 24 * 60 * 60

// _uploadsCSP makes uploaded files inert in the API's origin. Anyone who can log in can upload, so
// an .html or .svg file must not be able to run script with access to the admin's stored JWT: the
// sandbox directive gives it an opaque origin.
const _uploadsCSP = "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox"

// uploadsHandler serves Media Library files from dir under prefix, which must start with "/" and
// not end with one (e.g. "/uploads").
//
// It uses fasthttp's file server directly rather than app.Static because it needs SkipCache: the
// default handler keeps served files open for 10 s, so a deleted file goes on being served, and on
// Windows the delete itself fails while the handle is open. Fiber v2 does not expose the option.
func uploadsHandler(prefix, dir string) fiber.Handler {
	fs := &fasthttp.FS{
		Root:            dir,
		SkipCache:       true,
		AcceptByteRange: true,
		PathRewrite:     fasthttp.NewPathPrefixStripper(len(prefix)),
	}
	serve := fs.NewRequestHandler()

	return func(ctx *fiber.Ctx) error {
		serve(ctx.Context())

		if ctx.Response().StatusCode() == fiber.StatusOK || ctx.Response().StatusCode() == fiber.StatusPartialContent {
			ctx.Set(fiber.HeaderCacheControl, "public, max-age="+strconv.Itoa(_uploadsMaxAge))
		}

		// nosniff stops the browser from guessing a more dangerous type than the extension says.
		ctx.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		ctx.Set(fiber.HeaderContentSecurityPolicy, _uploadsCSP)

		return nil
	}
}
