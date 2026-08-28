package container

import (
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

// resolveInheritedAuth walks request → parent folders → collection, preferring drafts.
func resolveInheritedAuth(app *shared.App, collectionID, itemID string, requestAuth entity.Auth) entity.Auth {
	ws := app.Store.Workspace.GetSelectedWorkspace()
	if ws == nil {
		return entity.ResolveAuth([]entity.Auth{requestAuth})
	}
	var col *entity.Collection
	for i := range ws.Collections {
		if ws.Collections[i].Id == collectionID {
			col = &ws.Collections[i]
			break
		}
	}
	if col == nil {
		return entity.ResolveAuth([]entity.Auth{requestAuth})
	}

	drafts := app.Store.Draft
	chain := []entity.Auth{requestAuth}

	path := itemPath(col.Items, itemID)
	// path is [rootFolder?, ..., parentFolder?, itemID]; skip the item itself.
	for i := len(path) - 2; i >= 0; i-- {
		fid := path[i]
		if d, ok := drafts.GetFolderDraft(fid); ok {
			chain = append(chain, d.Auth)
			continue
		}
		if folder := findCollectionItem(col.Items, fid); folder != nil {
			chain = append(chain, folder.Auth)
		}
	}

	if d, ok := drafts.GetCollectionDraft(collectionID); ok {
		chain = append(chain, d.Auth)
	} else {
		chain = append(chain, col.Auth)
	}
	return entity.ResolveAuth(chain)
}

func itemPath(items []entity.CollectionItem, id string) []string {
	for i := range items {
		if items[i].Id == id {
			return []string{items[i].Id}
		}
		if sub := itemPath(items[i].Item, id); sub != nil {
			return append([]string{items[i].Id}, sub...)
		}
	}
	return nil
}

func applyRESTAuth(app *shared.App, req entity.RestRequest, collectionID, itemID string, requestAuth entity.Auth) entity.RestRequest {
	req.Auth = resolveInheritedAuth(app, collectionID, itemID, requestAuth)
	return req
}
