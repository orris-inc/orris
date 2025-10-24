package value_objects

import "fmt"

type ReadStatus string

const (
	ReadStatusUnread ReadStatus = "unread"
	ReadStatusRead   ReadStatus = "read"
)

var validReadStatuses = map[ReadStatus]bool{
	ReadStatusUnread: true,
	ReadStatusRead:   true,
}

func (s ReadStatus) String() string {
	return string(s)
}

func (s ReadStatus) IsValid() bool {
	return validReadStatuses[s]
}

func (s ReadStatus) IsUnread() bool {
	return s == ReadStatusUnread
}

func (s ReadStatus) IsRead() bool {
	return s == ReadStatusRead
}

func NewReadStatus(str string) (ReadStatus, error) {
	s := ReadStatus(str)
	if !s.IsValid() {
		return "", fmt.Errorf("invalid read status: %s", str)
	}
	return s, nil
}
