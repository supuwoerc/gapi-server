package model

import (
	"database/sql/driver"
	"time"

	"github.com/pkg/errors"
	"gorm.io/plugin/soft_delete"
)

type UpsertTime time.Time

func (c UpsertTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.DateTime)+len(`""`))
	b = append(b, '"')
	b = time.Time(c).AppendFormat(b, time.DateTime)
	b = append(b, '"')
	return b, nil
}

func (c *UpsertTime) UnmarshalJSON(bytes []byte) error {
	str := string(bytes)
	if str == "null" || str == `""` {
		*c = UpsertTime{} // 设置为零值
		return nil
	}
	if parse, err := time.ParseInLocation(`"`+time.DateTime+`"`, str, time.Local); err != nil {
		return err
	} else {
		*c = UpsertTime(parse)
		return nil
	}
}

func (c UpsertTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	t := time.Time(c)
	if t.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t, nil
}

func (c *UpsertTime) Scan(v interface{}) error {
	if v == nil {
		*c = UpsertTime{} // 处理 nil 值
		return nil
	}
	switch value := v.(type) {
	case time.Time:
		*c = UpsertTime(value)
		return nil
	case *time.Time:
		if value == nil {
			*c = UpsertTime{}
			return nil
		}
		*c = UpsertTime(*value)
		return nil
	case []byte:
		if len(value) == 0 {
			*c = UpsertTime{}
			return nil
		}
		if t, err := time.ParseInLocation(time.DateTime, string(value), time.Local); err != nil {
			return errors.Wrapf(err, "[UpsertTime] can not convert %v to timestamp", v)
		} else {
			*c = UpsertTime(t)
			return nil
		}
	case string:
		if value == "" {
			*c = UpsertTime{}
			return nil
		}
		if t, err := time.ParseInLocation(time.DateTime, value, time.Local); err != nil {
			return errors.Wrapf(err, "[UpsertTime] can not convert %v to timestamp", v)
		} else {
			*c = UpsertTime(t)
			return nil
		}
	default:
		return errors.Errorf("[UpsertTime] can not convert %v (type %T) to timestamp", v, v)
	}
}

// BaseModel is the common base for all GORM models.
type BaseModel struct {
	ID        uint                  `json:"id" gorm:"primarykey"`
	CreatedAt UpsertTime            `json:"created_at"`
	UpdatedAt UpsertTime            `json:"updated_at"`
	DeletedAt soft_delete.DeletedAt `json:"deleted_at,omitempty" gorm:"softDelete:milli;index"`
}
