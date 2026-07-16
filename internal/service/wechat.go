package service

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"

	"devtoolbox/internal/model"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

const defaultWeChatThemeID = "red"

type WeChatService struct {
	markdown goldmark.Markdown
	themes   []wechatTheme
}

type wechatTheme struct {
	ID     string
	Name   string
	Styles map[string]string
}

func NewWeChatService() *WeChatService {
	return &WeChatService{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(ghtml.WithHardWraps()),
		),
		themes: builtinWeChatThemes(),
	}
}

func (s *WeChatService) Themes() []model.WeChatTheme {
	out := make([]model.WeChatTheme, 0, len(s.themes))
	for _, theme := range s.themes {
		if theme.ID == defaultWeChatThemeID {
			out = append(out, model.WeChatTheme{ID: theme.ID, Name: theme.Name})
		}
	}
	for _, theme := range s.themes {
		if theme.ID != defaultWeChatThemeID {
			out = append(out, model.WeChatTheme{ID: theme.ID, Name: theme.Name})
		}
	}
	return out
}

func (s *WeChatService) Format(markdownText, themeID string) (model.WeChatFormatResp, error) {
	markdownText = strings.TrimSpace(markdownText)
	if markdownText == "" {
		return model.WeChatFormatResp{}, fmt.Errorf("markdown 内容不能为空")
	}

	var body bytes.Buffer
	if err := s.markdown.Convert([]byte(markdownText), &body); err != nil {
		return model.WeChatFormatResp{}, err
	}

	theme := s.findTheme(themeID)
	styledHTML := applyWeChatTheme(body.String(), theme)
	return model.WeChatFormatResp{
		HTML:     styledHTML,
		FullHTML: buildWeChatFullHTML(styledHTML, theme.Name),
		Theme:    theme.ID,
	}, nil
}

func (s *WeChatService) findTheme(themeID string) wechatTheme {
	if themeID == "" {
		themeID = defaultWeChatThemeID
	}
	for _, theme := range s.themes {
		if theme.ID == themeID {
			return theme
		}
	}
	for _, theme := range s.themes {
		if theme.ID == defaultWeChatThemeID {
			return theme
		}
	}
	return s.themes[0]
}

func applyWeChatTheme(rawHTML string, theme wechatTheme) string {
	out := rawHTML
	tags := []string{"h1", "h2", "h3", "h4", "h5", "h6", "p", "blockquote", "ul", "ol", "li", "pre", "code", "strong", "em", "a", "img", "hr", "table", "thead", "tbody", "tr", "th", "td"}
	for _, tag := range tags {
		style := theme.Styles[tag]
		if style == "" {
			continue
		}
		out = addInlineStyle(out, tag, style)
	}
	out = normalizeCodeBlockStyle(out)

	containerStyle := theme.Styles["container"]
	if containerStyle == "" {
		containerStyle = builtinWeChatThemes()[0].Styles["container"]
	}
	return `<section style="` + html.EscapeString(containerStyle) + `">` + "\n" + out + "\n</section>"
}

func addInlineStyle(input, tag, style string) string {
	pattern := regexp.MustCompile(`(?i)<` + tag + `(\s[^>]*)?>`)
	return pattern.ReplaceAllStringFunc(input, func(openTag string) string {
		if strings.Contains(strings.ToLower(openTag), " style=") {
			return openTag
		}
		if strings.HasSuffix(openTag, "/>") {
			return strings.TrimSuffix(openTag, "/>") + ` style="` + html.EscapeString(style) + `"/>`
		}
		return strings.TrimSuffix(openTag, ">") + ` style="` + html.EscapeString(style) + `">`
	})
}

func normalizeCodeBlockStyle(input string) string {
	pattern := regexp.MustCompile(`(?i)(<pre[^>]*>\s*<code)([^>]*) style="[^"]*"`)
	return pattern.ReplaceAllString(input, `$1$2 style="background: transparent; color: inherit; padding: 0; margin: 0; border-radius: 0; font-size: inherit; font-family: inherit;"`)
}

func buildWeChatFullHTML(bodyHTML, themeName string) string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(themeName) + `</title>
</head>
<body>
` + bodyHTML + `
</body>
</html>`
}

func builtinWeChatThemes() []wechatTheme {
	return []wechatTheme{
		{
			ID:   "default",
			Name: "默认清爽",
			Styles: map[string]string{
				"container":  "max-width: 677px; margin: 0 auto; padding: 24px 16px; color: #2f3440; font-size: 16px; line-height: 1.85; letter-spacing: 0.02em; font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;",
				"h1":         "margin: 8px 0 22px; padding-bottom: 12px; border-bottom: 2px solid #0f766e; color: #0f172a; font-size: 26px; line-height: 1.35; font-weight: 700;",
				"h2":         "margin: 32px 0 14px; padding-left: 12px; border-left: 4px solid #0f766e; color: #0f172a; font-size: 21px; line-height: 1.45; font-weight: 700;",
				"h3":         "margin: 24px 0 10px; color: #0f766e; font-size: 18px; line-height: 1.5; font-weight: 700;",
				"h4":         "margin: 20px 0 8px; color: #334155; font-size: 16px; font-weight: 700;",
				"h5":         "margin: 18px 0 8px; color: #334155; font-size: 15px; font-weight: 700;",
				"h6":         "margin: 16px 0 8px; color: #64748b; font-size: 14px; font-weight: 700;",
				"p":          "margin: 14px 0; color: #2f3440; font-size: 16px; line-height: 1.85;",
				"blockquote": "margin: 18px 0; padding: 12px 16px; border-left: 4px solid #99f6e4; background: #f0fdfa; color: #475569; border-radius: 8px;",
				"ul":         "margin: 14px 0; padding-left: 24px; color: #2f3440;",
				"ol":         "margin: 14px 0; padding-left: 24px; color: #2f3440;",
				"li":         "margin: 6px 0; line-height: 1.8;",
				"pre":        "margin: 18px 0; padding: 14px 16px; overflow-x: auto; background: #0f172a; border-radius: 10px; color: #e2e8f0; font-size: 13px; line-height: 1.75;",
				"code":       "padding: 2px 5px; margin: 0 2px; background: #eef2ff; color: #0f766e; border-radius: 4px; font-size: 90%; font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;",
				"a":          "color: #0f766e; text-decoration: none; border-bottom: 1px solid #99f6e4;",
				"img":        "display: block; max-width: 100%; height: auto; margin: 18px auto; border-radius: 10px;",
				"hr":         "margin: 28px 0; border: 0; border-top: 1px dashed #cbd5e1;",
				"table":      "width: 100%; margin: 18px 0; border-collapse: collapse; font-size: 14px;",
				"th":         "padding: 8px 10px; border: 1px solid #cbd5e1; background: #f8fafc; color: #0f172a; font-weight: 700;",
				"td":         "padding: 8px 10px; border: 1px solid #cbd5e1; color: #334155;",
			},
		},
		{
			ID:   "tech",
			Name: "技术文章",
			Styles: map[string]string{
				"container":  "max-width: 677px; margin: 0 auto; padding: 24px 16px; color: #263244; font-size: 16px; line-height: 1.82; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;",
				"h1":         "margin: 8px 0 24px; padding: 18px 20px; background: linear-gradient(135deg, #111827, #1e3a8a); color: #ffffff; border-radius: 14px; font-size: 25px; line-height: 1.35; font-weight: 800;",
				"h2":         "margin: 34px 0 14px; color: #1d4ed8; font-size: 21px; line-height: 1.45; font-weight: 800; border-bottom: 1px solid #bfdbfe; padding-bottom: 8px;",
				"h3":         "margin: 26px 0 10px; color: #111827; font-size: 18px; line-height: 1.5; font-weight: 700;",
				"h4":         "margin: 20px 0 8px; color: #334155; font-size: 16px; font-weight: 700;",
				"h5":         "margin: 18px 0 8px; color: #334155; font-size: 15px; font-weight: 700;",
				"h6":         "margin: 16px 0 8px; color: #64748b; font-size: 14px; font-weight: 700;",
				"p":          "margin: 14px 0; color: #263244; font-size: 16px; line-height: 1.82;",
				"blockquote": "margin: 18px 0; padding: 12px 16px; border-left: 4px solid #3b82f6; background: #eff6ff; color: #334155; border-radius: 8px;",
				"ul":         "margin: 14px 0; padding-left: 24px; color: #263244;",
				"ol":         "margin: 14px 0; padding-left: 24px; color: #263244;",
				"li":         "margin: 6px 0; line-height: 1.8;",
				"pre":        "margin: 18px 0; padding: 15px 16px; overflow-x: auto; background: #111827; border-radius: 12px; color: #dbeafe; font-size: 13px; line-height: 1.75;",
				"code":       "padding: 2px 6px; margin: 0 2px; background: #dbeafe; color: #1d4ed8; border-radius: 5px; font-size: 90%; font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;",
				"a":          "color: #2563eb; text-decoration: none; border-bottom: 1px solid #93c5fd;",
				"img":        "display: block; max-width: 100%; height: auto; margin: 18px auto; border-radius: 12px; box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);",
				"hr":         "margin: 28px 0; border: 0; border-top: 1px solid #bfdbfe;",
				"table":      "width: 100%; margin: 18px 0; border-collapse: collapse; font-size: 14px;",
				"th":         "padding: 8px 10px; border: 1px solid #bfdbfe; background: #eff6ff; color: #1e3a8a; font-weight: 700;",
				"td":         "padding: 8px 10px; border: 1px solid #bfdbfe; color: #334155;",
			},
		},
		{
			ID:   "red",
			Name: "红绯强调",
			Styles: map[string]string{
				"container":  "max-width: 677px; margin: 0 auto; padding: 24px 16px; color: #595959; font-size: 16px; line-height: 1.9; letter-spacing: 0.02em; font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif; background: #ffffff;",
				"h1":         "margin: 8px 0 24px; color: #1f2329; font-size: 25px; line-height: 1.35; font-weight: 800; letter-spacing: 0.01em;",
				"h2":         "margin: 34px 0 18px; padding: 1px 0 1px 12px; border-left: 5px solid #ff3b30; color: #1f2329; font-size: 19px; line-height: 1.55; font-weight: 800;",
				"h3":         "margin: 26px 0 12px; color: #1f2329; font-size: 17px; line-height: 1.55; font-weight: 800;",
				"h4":         "margin: 22px 0 10px; color: #1f2329; font-size: 16px; line-height: 1.55; font-weight: 700;",
				"h5":         "margin: 18px 0 8px; color: #1f2329; font-size: 15px; line-height: 1.55; font-weight: 700;",
				"h6":         "margin: 16px 0 8px; color: #595959; font-size: 14px; line-height: 1.55; font-weight: 700;",
				"p":          "margin: 15px 0; color: #595959; font-size: 16px; line-height: 1.9;",
				"blockquote": "margin: 20px 0; padding: 12px 16px; border-left: 5px solid #ff3b30; background: #fff5f5; color: #595959;",
				"ul":         "margin: 14px 0; padding-left: 24px; color: #595959;",
				"ol":         "margin: 14px 0; padding-left: 24px; color: #595959;",
				"li":         "margin: 7px 0; line-height: 1.85;",
				"pre":        "margin: 18px 0; padding: 14px 16px; overflow-x: auto; background: #2b2f36; color: #f5f5f5; font-size: 13px; line-height: 1.75; border-radius: 8px;",
				"code":       "padding: 2px 5px; margin: 0 2px; background: #fff1f0; color: #ff3b30; border-radius: 4px; font-size: 90%; font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;",
				"strong":     "color: #ff3b30; font-weight: 800;",
				"em":         "color: #ff3b30; font-style: normal; font-weight: 700;",
				"a":          "color: #ff3b30; text-decoration: none; border-bottom: 1px solid #ffccc7;",
				"img":        "display: block; max-width: 100%; height: auto; margin: 20px auto;",
				"hr":         "margin: 30px 0; border: 0; border-top: 1px solid #f1f1f1;",
				"table":      "width: 100%; margin: 18px 0; border-collapse: collapse; font-size: 14px;",
				"th":         "padding: 8px 10px; border: 1px solid #f0f0f0; background: #fff5f5; color: #1f2329; font-weight: 700;",
				"td":         "padding: 8px 10px; border: 1px solid #f0f0f0; color: #595959;",
			},
		},
	}
}
