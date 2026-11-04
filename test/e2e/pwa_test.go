//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// TestShareTarget opens the page the way the installed app's share target
// does (GET /?title=…&text=…&url=…): the YouTube link inside the shared text
// loads, its t= sets the start, and the address bar is reset to /.
func TestShareTarget(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, query, link, id, from string }{
		{"YouTube app (link in text)",
			"title=" + url.QueryEscape("Stub Video") + "&text=" + url.QueryEscape("Stub Video https://youtu.be/"+stubVideoID+"?si=AbC&t=8"),
			"https://youtu.be/" + stubVideoID + "?si=AbC&t=8", stubVideoID, "0:08"},
		{"browser (link in url)",
			"title=x&url=" + url.QueryEscape("https://www.youtube.com/watch?v=abcdefghijk"),
			"https://www.youtube.com/watch?v=abcdefghijk", "abcdefghijk", "0:00"},
		{"link in parentheses",
			"text=" + url.QueryEscape("watch this (https://youtu.be/ABCDEFGHIJK)."),
			"https://youtu.be/ABCDEFGHIJK", "ABCDEFGHIJK", "0:00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := newPage(t)
			p.do("the editor did not appear for a shared link",
				chromedp.Navigate(p.base+"/?"+tc.query),
				chromedp.WaitVisible(editor),
			)
			poll[bool](p, "the video length did not arrive", `document.getElementById("trimmer").dataset.ready === "true"`, 15*time.Second)
			if got := p.value("#url"); got != tc.link {
				t.Errorf("URL field = %q; want %q", got, tc.link)
			}
			if loc := eval[string](p, `location.pathname + location.search`); loc != "/" {
				t.Errorf("address bar = %q; want /", loc)
			}
			if from := p.value("#from"); from != tc.from {
				t.Errorf("start = %s; want %s", from, tc.from)
			}
			infos := p.fetches("/api/v1/video/info?")
			if len(infos) != 1 || !strings.Contains(infos[0].URL, "watch%3Fv%3D"+tc.id) {
				t.Fatalf("expected one info request for %s, got %+v", tc.id, infos)
			}
		})
	}

	t.Run("no YouTube link shared", func(t *testing.T) {
		t.Parallel()
		p := newPage(t)
		p.do("loading the page", chromedp.Navigate(p.base+"/?text="+url.QueryEscape("just some text https://example.com/x")), chromedp.WaitVisible(urlInput))
		if !p.hidden("editor") {
			t.Error("the editor opened although nothing YouTube was shared")
		}
	})
}

// TestFindYouTubeLink runs the page's shared-text parser in the browser.
func TestFindYouTubeLink(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.navigate()
	cases := map[string]string{ // text → link ("" = none)
		"https://youtu.be/dQw4w9WgXcQ":                                    "https://youtu.be/dQw4w9WgXcQ",
		"Never Gonna Give You Up https://youtu.be/dQw4w9WgXcQ?si=xYz":     "https://youtu.be/dQw4w9WgXcQ?si=xYz",
		"see https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=42s, it's fun": "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=42s",
		"<https://m.youtube.com/shorts/dQw4w9WgXcQ>":                      "https://m.youtube.com/shorts/dQw4w9WgXcQ",
		"two: https://example.com https://youtu.be/dQw4w9WgXcQ":           "https://youtu.be/dQw4w9WgXcQ",
		"https://vimeo.com/123":                                           "",
		"":                                                                "",
	}
	texts, err := json.Marshal(keys(cases))
	if err != nil {
		t.Fatal(err)
	}
	got, err := chromedp.Run(p.ctx, chromedp.Evaluate[map[string]string](fmt.Sprintf(`
		import("/static/scripts/youtube-url.js").then(({ findYouTubeLink }) =>
			Object.fromEntries(%s.map((text) => [text, findYouTubeLink(text) ?? ""])))`, texts),
		chromedp.EvalAwaitPromise))
	if err != nil {
		t.Fatal(err)
	}
	for text, want := range cases {
		if got[text] != want {
			t.Errorf("findYouTubeLink(%q) = %q; want %q", text, got[text], want)
		}
	}
}

// TestInstallable: Chrome parses the manifest (share target included), the
// service worker controls the page, and Chrome reports no installability
// errors other than the test's own incognito-like browser context.
func TestInstallable(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.navigate()

	scope := poll[string](p, "the service worker did not activate",
		`navigator.serviceWorker && navigator.serviceWorker.controller ? navigator.serviceWorker.controller.scriptURL : ""`, 15*time.Second)
	if !strings.HasSuffix(scope, "/sw.js") {
		t.Errorf("service worker = %q; want /sw.js", scope)
	}

	manifest, err := chromedp.Call(p.ctx, cdppage.GetAppManifest, cdppage.GetAppManifestParams{})
	if err != nil {
		t.Fatalf("getting the manifest: %v", err)
	}
	if !strings.HasSuffix(manifest.URL, "/manifest.webmanifest") {
		t.Fatalf("manifest URL = %q", manifest.URL)
	}
	for _, e := range manifest.Errors {
		t.Errorf("manifest error (critical %d) at %d:%d: %s", e.Critical, e.Line, e.Column, e.Message)
	}
	if !strings.Contains(manifest.Data, `"share_target"`) {
		t.Error("the parsed manifest has no share_target")
	}

	installability, err := chromedp.Call(p.ctx, cdppage.GetInstallabilityErrors, cdp.Empty{})
	if err != nil {
		t.Fatalf("installability: %v", err)
	}
	for _, e := range installability.InstallabilityErrors {
		if e.ErrorID != "in-incognito" { // every test tab is a fresh browser context
			t.Errorf("installability error %s %+v", e.ErrorID, e.ErrorArguments)
		}
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
