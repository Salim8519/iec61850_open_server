package goosedom

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// blockFields are the block keys copied into the link map, in Python order.
var blockFields = [...]string{"publisher", "ld_inst", "name", "gocb_ref", "go_id", "dataset", "dataset_ref",
	"conf_rev", "app_id", "mac", "status", "label"}

// linkID is _id: the first 16 hex digits of sha256 over "|".join(str(part)).
func linkID(parts ...any) string {
	texts := make([]string, len(parts))
	for index, part := range parts {
		texts[index] = pyv.PyStr(part)
	}
	sum := sha256.Sum256([]byte(strings.Join(texts, "|")))
	return hex.EncodeToString(sum[:])[:16]
}

// fcdaNode is prefix + lnClass + lnInst and the "doName.daName" path of an FCDA.
func fcdaNode(fcda *pyjson.Object) (string, string) {
	node := strOr(fcda, "prefix", "") + strOr(fcda, "lnClass", "") + strOr(fcda, "lnInst", "")
	var parts []string
	for _, key := range []string{"doName", "daName"} {
		if part := pyv.GetOr(fcda, key, ""); pyv.Truthy(part) {
			parts = append(parts, pyv.PyStr(part))
		}
	}
	return node, strings.Join(parts, ".")
}

// memberText is _member: "ldInst/node.path [fc]".
func memberText(fcda *pyjson.Object) string {
	node, path := fcdaNode(fcda)
	return strOr(fcda, "ldInst", "") + "/" + node + "." + path + " [" + strOr(fcda, "fc", "") + "]"
}

// memberRef is _member_ref: the data reference of a dataset member.
func memberRef(block, fcda *pyjson.Object) string {
	domain := ""
	if ref := pyv.Get(block, "dataset_ref"); pyv.Truthy(ref) {
		domain = strings.SplitN(pyv.PyStr(ref), "/", 2)[0]
	}
	ldInst := pyv.Get(block, "ld_inst")
	if !pyv.Equal(pyv.GetOr(fcda, "ldInst", ldInst), ldInst) || domain == "" {
		domain = pyv.PyStr(pyv.Get(block, "publisher")) + strOr(fcda, "ldInst", "")
	}
	node, path := fcdaNode(fcda)
	return domain + "/" + node + "." + path
}

// blockView is _block: the block fields plus its members and member references.
func blockView(block *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range blockFields {
		result.Set(key, pyv.Get(block, key))
	}
	members, refs := []any{}, []any{}
	for _, fcda := range pyv.Objects(pyv.GetList(block, "fcdas")) {
		members = append(members, memberText(fcda))
		refs = append(refs, memberRef(block, fcda))
	}
	return result.Set("members", members).Set("member_refs", refs)
}

// names is _names: "publisher/name" of the first six blocks and "and N more".
func names(blocks []*pyjson.Object) string {
	const limit = 6
	texts := make([]string, len(blocks))
	for index, block := range blocks {
		texts[index] = pyv.PyStr(pyv.Get(block, "publisher")) + "/" + pyv.PyStr(pyv.Get(block, "name"))
	}
	if len(texts) <= limit {
		return strings.Join(texts, ", ")
	}
	return strings.Join(texts[:limit], ", ") + " and " + strconv.Itoa(len(texts)-limit) + " more"
}
