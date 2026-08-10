package store

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
)

type ICollectionStore interface {
	FetchCollectionList()
	GetIsFetching() *binding.Bool

	ListCollectionItems() *binding.UntypedList
	SelectCollectionItem(id string)
	GetItem(id string) *entity.Collection
	GetCollectionDataItem(item binding.DataItem) *entity.Collection
}
