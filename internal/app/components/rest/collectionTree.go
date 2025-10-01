package rest

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/goose/internal/app/entity"
)

// TreeData структура для данных дерева
type TreeData struct {
	collections []entity.Collection
}

// TreeAdapter адаптер для виджета дерева
type TreeAdapter struct {
	data *TreeData
}

type CollectionTree struct {
	*widget.Tree
	adapter *TreeAdapter
	allUIDs map[widget.TreeNodeID]bool
}

// NewCollectionTree Функция для создания и отображения дерева
func NewCollectionTree(collections []entity.Collection, onSelect func(itemId string)) *CollectionTree {
	data := &TreeData{collections: collections}
	ct := &CollectionTree{
		adapter: &TreeAdapter{data: data},
		allUIDs: make(map[widget.TreeNodeID]bool),
	}

	tree := widget.NewTree(ct.childIDs, ct.isBranch, ct.createNode, ct.updateNode)
	ct.Tree = tree
	tree.OpenAllBranches()

	// TODO : fix rendering
	refreshAllItems := func() {
		for uid := range ct.allUIDs {
			tree.RefreshItem(uid)
		}
	}
	refreshItem := func(uid widget.TreeNodeID) {
		tree.RefreshItem(uid)
	}
	tree.OnBranchOpened = refreshItem
	tree.OnBranchClosed = refreshItem
	tree.OnUnselected = func(uid widget.TreeNodeID) {
		refreshAllItems()
	}
	tree.OnSelected = func(uid widget.TreeNodeID) {
		refreshAllItems()
		nodeType, itemID, parentID := parseID(uid)
		fmt.Printf("Selected: %s - %s (Parent: %s)\n", nodeType, itemID, parentID)
		if onSelect != nil {
			onSelect(itemID)
		}
		//refreshItem(uid)
	}

	return ct
}

// generateID создает уникальный ID для узла дерева
func generateID(nodeType, id, parentID string) string {
	if parentID == "" {
		return nodeType + ":" + id
	}
	return nodeType + ":" + id + ":" + parentID
}

// parseID парсит ID узла дерева
func parseID(id string) (nodeType, itemID, parentID string) {
	// Формат: "type:id" или "type:id:parentID"
	for i := 0; i < len(id); i++ {
		if id[i] == ':' {
			nodeType = id[:i]
			remaining := id[i+1:]
			for j := 0; j < len(remaining); j++ {
				if remaining[j] == ':' {
					itemID = remaining[:j]
					parentID = remaining[j+1:]
					return
				}
			}
			itemID = remaining
			return
		}
	}
	return "", id, ""
}

func (ct *CollectionTree) UpdateCollections(collections []entity.Collection) {
	fmt.Printf("🔄 Updating tree with %d collections\n", len(collections))

	ct.adapter.data.collections = make([]entity.Collection, len(collections))
	copy(ct.adapter.data.collections, collections)

	// Полное перестроение дерева
	ct.Tree.Refresh()
	ct.Refresh()
}

// Вспомогательная функция для подсчета элементов
func (ct *CollectionTree) countTotalItems(collections []entity.Collection) int {
	count := 0
	for _, col := range collections {
		count += ct.countItemsInCollection(col.Item)
	}
	return count
}

func (ct *CollectionTree) countItemsInCollection(items []entity.CollectionItem) int {
	count := len(items)
	for _, item := range items {
		count += ct.countItemsInCollection(item.Item)
	}
	return count
}

// возвращает дочерние узлы для родительского узла
func (ct *CollectionTree) childIDs(id string) []string {
	if id == "" {
		// Корневые узлы - коллекции
		var children []string
		for _, col := range ct.adapter.data.collections {
			uid := generateID("collection", col.Id, "")
			children = append(children, uid)
			ct.allUIDs[uid] = true
		}
		return children
	}

	nodeType, itemID, _ := parseID(id)
	switch nodeType {
	case "collection":
		// Дочерние узлы коллекции - элементы
		for _, col := range ct.adapter.data.collections {
			if col.Id == itemID {
				var children []string
				for _, item := range col.Item {
					children = append(children, generateID("item", item.Id, col.Id))
				}
				return children
			}
		}
	case "item":
		// Дочерние узлы элемента - вложенные элементы (рекурсивно)
		for _, col := range ct.adapter.data.collections {
			if found := ct.findItemChildren(col.Item, itemID); found != nil {
				return found
			}
		}
	}

	return nil
}

// вспомогательная функция для рекурсивного поиска дочерних элементов
func (ct *CollectionTree) findItemChildren(items []entity.CollectionItem, targetID string) []string {
	for _, item := range items {
		if item.Id == targetID {
			var children []string
			for _, subItem := range item.Item {
				children = append(children, generateID("item", subItem.Id, item.Id))
			}
			return children
		}
		// Рекурсивный поиск во вложенных элементах
		if children := ct.findItemChildren(item.Item, targetID); children != nil {
			return children
		}
	}
	return nil
}

// создает виджет для узла дерева
func (ct *CollectionTree) createNode(_ bool) fyne.CanvasObject {
	return widget.NewLabel("")
}

// проверяет, является ли узел веткой (имеет дочерние элементы)
func (ct *CollectionTree) isBranch(id string) bool {
	return len(ct.childIDs(id)) > 0
}

// обновляет содержимое узла
func (ct *CollectionTree) updateNode(id string, _ bool, node fyne.CanvasObject) {
	label := node.(*widget.Label)
	nodeType, itemID, _ := parseID(id)

	var text string
	var found bool

	switch nodeType {
	case "collection":
		for _, col := range ct.adapter.data.collections {
			if col.Id == itemID {
				text = col.Name
				found = true
				break
			}
		}
	case "item":
		found = ct.findItemText(&text, ct.adapter.data.collections, itemID)
	default:
		text = "Неизвестный тип: " + nodeType
	}

	if !found {
		text = "Не найден: " + itemID
	}

	label.SetText(text)
}

// Вспомогательная функция для рекурсивного поиска и обновления текста элемента
func (ct *CollectionTree) findItemText(text *string, collections []entity.Collection, targetID string) bool {
	for _, col := range collections {
		if ct.findItemInCollection(text, col.Item, targetID) {
			return true
		}
	}
	return false
}

// Вспомогательная функция для поиска в коллекции
func (ct *CollectionTree) findItemInCollection(text *string, items []entity.CollectionItem, targetID string) bool {
	for _, item := range items {
		if item.Id == targetID {
			if item.Request != nil {
				*text = "[" + string(item.Request.Method) + "] " + item.Name
			} else {
				*text = "📁 " + item.Name
			}

			return true
		}
		// Рекурсивный поиск во вложенных элементах
		if ct.findItemInCollection(text, item.Item, targetID) {
			return true
		}
	}
	return false
}
