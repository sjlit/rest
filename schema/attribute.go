package schema

import (
	"database/sql/driver"
	"encoding/json"
)

type (
	LiveValue struct {
		Enable      bool     `json:"enable"`
		Type        string   `json:"type"`
		Url         string   `json:"url"`
		Method      string   `json:"method"`
		Body        string   `json:"body"`
		ContentType string   `json:"content_type"`
		Columns     []string `json:"columns"`
	}

	EnumValue struct {
		Label string `json:"label"`
		Value string `json:"value"`
		Color string `json:"color"`
	}

	VisibleCondition struct {
		Column string `json:"column"`
		Values []any  `json:"values"`
	}

	DropdownOptions struct {
		Created      bool `json:"created,omitempty"`
		Searchable   bool `json:"searchable,omitempty"`
		Filterable   bool `json:"filterable,omitempty"`
		Autocomplete bool `json:"autocomplete,omitempty"`
		DefaultFirst bool `json:"default_first,omitempty"`
		Multiple     bool `json:"multiple,omitempty"`
		CollapseTags bool `json:"collapse_tags,omitempty"`
	}

	Attribute struct {
		Match           string             `json:"match"`                 //匹配模式
		Tag             string             `json:"tag,omitempty"`         //字段标签
		DefaultValue    string             `json:"default_value"`         //默认值
		Readonly        []string           `json:"readonly"`              //只读场景
		Disable         []string           `json:"disable"`               //禁用场景
		Visible         []VisibleCondition `json:"visible"`               //显示场景
		Invisible       bool               `json:"invisible"`             //不可见的字段，表示在UI界面看不到这个字段，但是这个字段需要
		EndOfNow        bool               `json:"end_of_now"`            //最大时间为当前时间
		TimeSearchRange string             `json:"time_search_range"`     //默认搜索的范围, 在时间字段里面可配置, 有 day 和 method
		Values          []EnumValue        `json:"values"`                //枚举的值
		Live            LiveValue          `json:"live"`                  //延时加载配置
		UploadUrl       string             `json:"upload_url,omitempty"`  //上传地址
		Icon            string             `json:"icon,omitempty"`        //显示图标
		Sort            bool               `json:"sort"`                  //是否允许排序
		Suffix          string             `json:"suffix,omitempty"`      //追加内容
		Tooltip         string             `json:"tooltip,omitempty"`     //字段提示信息
		DropdownOptions *DropdownOptions   `json:"dropdown,omitempty"`    //下拉选项
		Description     string             `json:"description,omitempty"` //字段说明信息
	}
)

// Scan implements the Scanner interface.
func (n *Attribute) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch s := value.(type) {
	case string:
		return json.Unmarshal([]byte(s), n)
	case []byte:
		return json.Unmarshal(s, n)
	}
	return ErrUnsupportType
}

// Value implements the driver Valuer interface.
func (n Attribute) Value() (driver.Value, error) {
	return json.Marshal(n)
}
