package schema

import "errors"

var (
	ErrUnsupportType = errors.New("database type unsupported")
)

type (
	Schema struct {
		Id         uint64    `json:"id" gorm:"primary_key"`
		CreatedAt  int64     `json:"created_at" gorm:"autoCreateTime"`                             //创建时间
		UpdatedAt  int64     `json:"updated_at" gorm:"autoUpdateTime"`                             //更新时间
		TenantID   string    `json:"tenant_id" gorm:"column:tenant_id;type:char(60);index"`        //域
		ModuleName string    `json:"module_name" gorm:"column:module_name;type:varchar(60);index"` //模块名称
		TableName  string    `json:"table_name" gorm:"column:table_name;type:varchar(120);index"`  //表名称
		Enable     uint8     `json:"enable" gorm:"column:enable"`                                  //是否启用
		Column     string    `json:"column" gorm:"type:varchar(120)"`                              //字段名称
		Label      string    `json:"label" gorm:"type:varchar(120)"`                               //显示名称
		Type       string    `json:"type" gorm:"type:varchar(120)"`                                //字段类型
		Format     string    `json:"format" gorm:"type:varchar(120)"`                              //字段格式
		Native     uint8     `json:"native"`                                                       //原生字段
		PrimaryKey uint8     `json:"primary_key"`                                                  //是否为主键
		Expression string    `json:"expression" gorm:"type:varchar(526)"`                          //计算规则
		Scenarios  Scenarios `json:"scenarios" gorm:"type:varchar(120)"`                           //场景
		Rules      Rule      `json:"rules" gorm:"type:varchar(2048)"`                              //字段规则
		Attributes Attribute `json:"attributes" gorm:"type:varchar(4096)"`                         //字段属性
		Position   int       `json:"position"`                                                     //字段排序位置
	}
)
