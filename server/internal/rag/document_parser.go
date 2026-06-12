// Package rag 实现自建 RAG 检索引擎。
//
// document_parser.go 实现多格式文档解析器。
//
// 支持格式：PDF / DOCX / MD / TXT
//
// 为什么使用 ledongthuc/pdf 而非其他 PDF 库：
// ledongthuc/pdf 是纯 Go 实现，无 CGO 依赖，交叉编译友好。
// 虽然高级功能（表格、图片）不如 poppler，但 MVP 阶段仅需文本提取即可。
//
// DOCX 解析使用 Go 标准库 archive/zip + encoding/xml，
// 直接解析 OOXML 格式中的 w:t（文本节点），
// 不依赖任何第三方库。
package rag

import (
	"fmt"
	"io"
	"strings"

	"archive/zip"
	"bytes"
	"encoding/xml"

	"github.com/ledongthuc/pdf"
)

// DocParser 多格式文档解析器。
type DocParser struct{}

// NewDocParser 创建文档解析器实例。
func NewDocParser() *DocParser {
	return &DocParser{}
}

// Parse 根据文件类型解析文档并返回纯文本内容。
//
// reader 在解析完成后不会关闭，由调用方负责关闭。
// fileType 支持：pdf / docx / md / txt（大小写不敏感）。
func (p *DocParser) Parse(reader io.Reader, fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "txt":
		return p.parseTxt(reader)
	case "md", "markdown":
		return p.parseTxt(reader) // MD 本质是纯文本，直接返回
	case "docx":
		return p.parseDocx(reader)
	case "pdf":
		return p.parsePDF(reader)
	default:
		return "", fmt.Errorf("不支持的文件类型: %s（支持的格式：pdf / docx / md / txt）", fileType)
	}
}

// parseTxt 读取纯文本/Markdown 文件。
//
// TODO: io.ReadAll 无大小限制，恶意大文件可导致 OOM。应使用 io.LimitReader 设置上限（如 100MB）。
func (p *DocParser) parseTxt(reader io.Reader) (string, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取文本文件失败: %w", err)
	}
	return string(b), nil
}

// parsePDF 解析 PDF 文件，逐页提取文本。
func (p *DocParser) parsePDF(reader io.Reader) (string, error) {
	// ledongthuc/pdf 需要 ReadAt 接口和文件大小
	// 对于流式 reader，先读入内存
	b, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取 PDF 文件失败: %w", err)
	}

	pdfReader, err := pdf.NewReader(strings.NewReader(string(b)), int64(len(b)))
	if err != nil {
		return "", fmt.Errorf("打开 PDF 失败: %w", err)
	}

	var buf strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // 跳过无法解析的页面
		}
		buf.WriteString(text)
		buf.WriteByte('\n')
	}
	return strings.TrimSpace(buf.String()), nil
}

// parseDocx 解析 DOCX (OOXML) 文件，提取 w:t 文本节点。
//
// DOCX 文件是一个 ZIP 压缩包，其中 word/document.xml 包含正文内容。
// 提取所有 <w:t> 元素中的文本，跳过其他元素（如格式、图片信息）。
func (p *DocParser) parseDocx(reader io.Reader) (string, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取 DOCX 文件失败: %w", err)
	}
	return parseDocxFromBytes(b)
}

// parseDocxFromBytes 从字节数组中解析 DOCX 内容。
//
// 流程：解压 ZIP → 读取 word/document.xml → 解析 XML → 提取 w:t 文本。
func parseDocxFromBytes(data []byte) (string, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打开 DOCX ZIP 失败: %w", err)
	}

	var documentXML []byte
	for _, f := range zipReader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("打开 DOCX document.xml 失败: %w", err)
			}
			documentXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("读取 DOCX document.xml 失败: %w", err)
			}
			break
		}
	}

	if documentXML == nil {
		return "", fmt.Errorf("DOCX 中未找到 word/document.xml")
	}

	return extractDocxText(documentXML)
}

// docxDocument DOCX document.xml 的 XML 结构（仅提取 w:t 内容）。
//
// 使用命名空间 URL 匹配 OOXML 元素（<w:document xmlns:w="...">），
// Go encoding/xml 以完整 URL 形式写在 tag 中即可匹配 namespaced 元素。
type docxDocument struct {
	XMLName xml.Name       `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main document"`
	Body    docxBody       `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main body"`
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main r"`
}

type docxRun struct {
	Text string `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main t"`
}

// extractDocxText 从 DOCX XML 中提取所有 <w:t> 文本。
func extractDocxText(xmlData []byte) (string, error) {
	var doc docxDocument
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return "", fmt.Errorf("解析 DOCX XML 失败: %w", err)
	}

	var buf strings.Builder
	for _, para := range doc.Body.Paragraphs {
		for _, run := range para.Runs {
			buf.WriteString(run.Text)
		}
		buf.WriteByte('\n')
	}
	return strings.TrimSpace(buf.String()), nil
}
