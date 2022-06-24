// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.1
// source: proto/store.proto

package store

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Protobuf defintions take inspiration from rqlite
type Parameter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Value:
	//	*Parameter_I
	//	*Parameter_D
	//	*Parameter_B
	//	*Parameter_Y
	//	*Parameter_S
	Value isParameter_Value `protobuf_oneof:"value"`
	Name  string            `protobuf:"bytes,6,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *Parameter) Reset() {
	*x = Parameter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Parameter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Parameter) ProtoMessage() {}

func (x *Parameter) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Parameter.ProtoReflect.Descriptor instead.
func (*Parameter) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{0}
}

func (m *Parameter) GetValue() isParameter_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *Parameter) GetI() int64 {
	if x, ok := x.GetValue().(*Parameter_I); ok {
		return x.I
	}
	return 0
}

func (x *Parameter) GetD() float64 {
	if x, ok := x.GetValue().(*Parameter_D); ok {
		return x.D
	}
	return 0
}

func (x *Parameter) GetB() bool {
	if x, ok := x.GetValue().(*Parameter_B); ok {
		return x.B
	}
	return false
}

func (x *Parameter) GetY() []byte {
	if x, ok := x.GetValue().(*Parameter_Y); ok {
		return x.Y
	}
	return nil
}

func (x *Parameter) GetS() string {
	if x, ok := x.GetValue().(*Parameter_S); ok {
		return x.S
	}
	return ""
}

func (x *Parameter) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type isParameter_Value interface {
	isParameter_Value()
}

type Parameter_I struct {
	I int64 `protobuf:"zigzag64,1,opt,name=i,proto3,oneof"`
}

type Parameter_D struct {
	D float64 `protobuf:"fixed64,2,opt,name=d,proto3,oneof"`
}

type Parameter_B struct {
	B bool `protobuf:"varint,3,opt,name=b,proto3,oneof"`
}

type Parameter_Y struct {
	Y []byte `protobuf:"bytes,4,opt,name=y,proto3,oneof"`
}

type Parameter_S struct {
	S string `protobuf:"bytes,5,opt,name=s,proto3,oneof"`
}

func (*Parameter_I) isParameter_Value() {}

func (*Parameter_D) isParameter_Value() {}

func (*Parameter_B) isParameter_Value() {}

func (*Parameter_Y) isParameter_Value() {}

func (*Parameter_S) isParameter_Value() {}

type Statement struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Sql    string       `protobuf:"bytes,1,opt,name=sql,proto3" json:"sql,omitempty"`
	Params []*Parameter `protobuf:"bytes,2,rep,name=params,proto3" json:"params,omitempty"`
}

func (x *Statement) Reset() {
	*x = Statement{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Statement) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Statement) ProtoMessage() {}

func (x *Statement) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Statement.ProtoReflect.Descriptor instead.
func (*Statement) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{1}
}

func (x *Statement) GetSql() string {
	if x != nil {
		return x.Sql
	}
	return ""
}

func (x *Statement) GetParams() []*Parameter {
	if x != nil {
		return x.Params
	}
	return nil
}

type Request struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Transaction bool         `protobuf:"varint,1,opt,name=transaction,proto3" json:"transaction,omitempty"`
	Statements  []*Statement `protobuf:"bytes,2,rep,name=statements,proto3" json:"statements,omitempty"`
}

func (x *Request) Reset() {
	*x = Request{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request) ProtoMessage() {}

func (x *Request) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request.ProtoReflect.Descriptor instead.
func (*Request) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{2}
}

func (x *Request) GetTransaction() bool {
	if x != nil {
		return x.Transaction
	}
	return false
}

func (x *Request) GetStatements() []*Statement {
	if x != nil {
		return x.Statements
	}
	return nil
}

// Same fields as request for now, but in the future it might change.
type QueryReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request *Request `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
}

func (x *QueryReq) Reset() {
	*x = QueryReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryReq) ProtoMessage() {}

func (x *QueryReq) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryReq.ProtoReflect.Descriptor instead.
func (*QueryReq) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{3}
}

func (x *QueryReq) GetRequest() *Request {
	if x != nil {
		return x.Request
	}
	return nil
}

type Values struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Params []*Parameter `protobuf:"bytes,1,rep,name=params,proto3" json:"params,omitempty"`
}

func (x *Values) Reset() {
	*x = Values{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Values) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Values) ProtoMessage() {}

func (x *Values) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Values.ProtoReflect.Descriptor instead.
func (*Values) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{4}
}

func (x *Values) GetParams() []*Parameter {
	if x != nil {
		return x.Params
	}
	return nil
}

type QueryRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Columns []string  `protobuf:"bytes,1,rep,name=columns,proto3" json:"columns,omitempty"` // The names of the columns
	Types   []string  `protobuf:"bytes,2,rep,name=types,proto3" json:"types,omitempty"`     // The types of the values
	Values  []*Values `protobuf:"bytes,3,rep,name=values,proto3" json:"values,omitempty"`
	Error   string    `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *QueryRes) Reset() {
	*x = QueryRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryRes) ProtoMessage() {}

func (x *QueryRes) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryRes.ProtoReflect.Descriptor instead.
func (*QueryRes) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{5}
}

func (x *QueryRes) GetColumns() []string {
	if x != nil {
		return x.Columns
	}
	return nil
}

func (x *QueryRes) GetTypes() []string {
	if x != nil {
		return x.Types
	}
	return nil
}

func (x *QueryRes) GetValues() []*Values {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *QueryRes) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type ExecRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LastInsertId int64  `protobuf:"varint,1,opt,name=last_insert_id,json=lastInsertId,proto3" json:"last_insert_id,omitempty"`
	RowsAffected int64  `protobuf:"varint,2,opt,name=rows_affected,json=rowsAffected,proto3" json:"rows_affected,omitempty"`
	Error        string `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *ExecRes) Reset() {
	*x = ExecRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecRes) ProtoMessage() {}

func (x *ExecRes) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecRes.ProtoReflect.Descriptor instead.
func (*ExecRes) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{6}
}

func (x *ExecRes) GetLastInsertId() int64 {
	if x != nil {
		return x.LastInsertId
	}
	return 0
}

func (x *ExecRes) GetRowsAffected() int64 {
	if x != nil {
		return x.RowsAffected
	}
	return 0
}

func (x *ExecRes) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type StoreExecResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Results []*ExecRes `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *StoreExecResponse) Reset() {
	*x = StoreExecResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreExecResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreExecResponse) ProtoMessage() {}

func (x *StoreExecResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreExecResponse.ProtoReflect.Descriptor instead.
func (*StoreExecResponse) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{7}
}

func (x *StoreExecResponse) GetResults() []*ExecRes {
	if x != nil {
		return x.Results
	}
	return nil
}

type StoreQueryResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Results []*QueryRes `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *StoreQueryResponse) Reset() {
	*x = StoreQueryResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreQueryResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreQueryResponse) ProtoMessage() {}

func (x *StoreQueryResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreQueryResponse.ProtoReflect.Descriptor instead.
func (*StoreQueryResponse) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{8}
}

func (x *StoreQueryResponse) GetResults() []*QueryRes {
	if x != nil {
		return x.Results
	}
	return nil
}

type ExecStringReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Exec string `protobuf:"bytes,1,opt,name=exec,proto3" json:"exec,omitempty"`
}

func (x *ExecStringReq) Reset() {
	*x = ExecStringReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecStringReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecStringReq) ProtoMessage() {}

func (x *ExecStringReq) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecStringReq.ProtoReflect.Descriptor instead.
func (*ExecStringReq) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{9}
}

func (x *ExecStringReq) GetExec() string {
	if x != nil {
		return x.Exec
	}
	return ""
}

type QueryStringReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Query string `protobuf:"bytes,1,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *QueryStringReq) Reset() {
	*x = QueryStringReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_store_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryStringReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryStringReq) ProtoMessage() {}

func (x *QueryStringReq) ProtoReflect() protoreflect.Message {
	mi := &file_proto_store_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryStringReq.ProtoReflect.Descriptor instead.
func (*QueryStringReq) Descriptor() ([]byte, []int) {
	return file_proto_store_proto_rawDescGZIP(), []int{10}
}

func (x *QueryStringReq) GetQuery() string {
	if x != nil {
		return x.Query
	}
	return ""
}

var File_proto_store_proto protoreflect.FileDescriptor

var file_proto_store_proto_rawDesc = []byte{
	0x0a, 0x11, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x05, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x22, 0x78, 0x0a, 0x09, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x12, 0x0e, 0x0a, 0x01, 0x69, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x12, 0x48, 0x00, 0x52, 0x01, 0x69, 0x12, 0x0e, 0x0a, 0x01, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x01, 0x48, 0x00, 0x52, 0x01, 0x64, 0x12, 0x0e, 0x0a, 0x01, 0x62, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x08, 0x48, 0x00, 0x52, 0x01, 0x62, 0x12, 0x0e, 0x0a, 0x01, 0x79, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0c, 0x48, 0x00, 0x52, 0x01, 0x79, 0x12, 0x0e, 0x0a, 0x01, 0x73, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x00, 0x52, 0x01, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x42, 0x07, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x22, 0x47, 0x0a, 0x09, 0x53, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e,
	0x74, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x71, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x73, 0x71, 0x6c, 0x12, 0x28, 0x0a, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x65, 0x74, 0x65, 0x72, 0x52, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x22, 0x5d, 0x0a,
	0x07, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x0b, 0x74, 0x72, 0x61, 0x6e,
	0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x74,
	0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x30, 0x0a, 0x0a, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10,
	0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74,
	0x52, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x34, 0x0a, 0x08,
	0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x12, 0x28, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x73, 0x74, 0x6f, 0x72,
	0x65, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x22, 0x32, 0x0a, 0x06, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x28, 0x0a, 0x06,
	0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x73,
	0x74, 0x6f, 0x72, 0x65, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x52, 0x06,
	0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x22, 0x77, 0x0a, 0x08, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52,
	0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73, 0x12, 0x14, 0x0a, 0x05,
	0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x12, 0x25, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x73, 0x52, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22,
	0x6a, 0x0a, 0x07, 0x45, 0x78, 0x65, 0x63, 0x52, 0x65, 0x73, 0x12, 0x24, 0x0a, 0x0e, 0x6c, 0x61,
	0x73, 0x74, 0x5f, 0x69, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x0c, 0x6c, 0x61, 0x73, 0x74, 0x49, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x49, 0x64,
	0x12, 0x23, 0x0a, 0x0d, 0x72, 0x6f, 0x77, 0x73, 0x5f, 0x61, 0x66, 0x66, 0x65, 0x63, 0x74, 0x65,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x72, 0x6f, 0x77, 0x73, 0x41, 0x66, 0x66,
	0x65, 0x63, 0x74, 0x65, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22, 0x3d, 0x0a, 0x11, 0x53,
	0x74, 0x6f, 0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x28, 0x0a, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x0e, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x52, 0x65,
	0x73, 0x52, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x22, 0x3f, 0x0a, 0x12, 0x53, 0x74,
	0x6f, 0x72, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x29, 0x0a, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x0f, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52,
	0x65, 0x73, 0x52, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x22, 0x23, 0x0a, 0x0d, 0x45,
	0x78, 0x65, 0x63, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x12, 0x12, 0x0a, 0x04,
	0x65, 0x78, 0x65, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x65, 0x78, 0x65, 0x63,
	0x22, 0x26, 0x0a, 0x0e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x52,
	0x65, 0x71, 0x12, 0x14, 0x0a, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x32, 0xf8, 0x01, 0x0a, 0x05, 0x53, 0x74, 0x6f,
	0x72, 0x65, 0x12, 0x35, 0x0a, 0x07, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x12, 0x0e, 0x2e,
	0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e,
	0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x35, 0x0a, 0x05, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x12, 0x0f, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79,
	0x52, 0x65, 0x71, 0x1a, 0x19, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x53, 0x74, 0x6f, 0x72,
	0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x12, 0x3e, 0x0a, 0x0a, 0x45, 0x78, 0x65, 0x63, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x12, 0x14,
	0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x53, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x52, 0x65, 0x71, 0x1a, 0x18, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x53, 0x74, 0x6f,
	0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x12, 0x41, 0x0a, 0x0b, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x12,
	0x15, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x1a, 0x19, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x53,
	0x74, 0x6f, 0x72, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x00, 0x42, 0x20, 0x5a, 0x1e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x6e, 0x69, 0x72, 0x65, 0x6f, 0x2f, 0x64, 0x69, 0x73, 0x74, 0x73, 0x71, 0x6c, 0x2f,
	0x73, 0x74, 0x6f, 0x72, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_store_proto_rawDescOnce sync.Once
	file_proto_store_proto_rawDescData = file_proto_store_proto_rawDesc
)

func file_proto_store_proto_rawDescGZIP() []byte {
	file_proto_store_proto_rawDescOnce.Do(func() {
		file_proto_store_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_store_proto_rawDescData)
	})
	return file_proto_store_proto_rawDescData
}

var file_proto_store_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_proto_store_proto_goTypes = []interface{}{
	(*Parameter)(nil),          // 0: store.Parameter
	(*Statement)(nil),          // 1: store.Statement
	(*Request)(nil),            // 2: store.Request
	(*QueryReq)(nil),           // 3: store.QueryReq
	(*Values)(nil),             // 4: store.Values
	(*QueryRes)(nil),           // 5: store.QueryRes
	(*ExecRes)(nil),            // 6: store.ExecRes
	(*StoreExecResponse)(nil),  // 7: store.StoreExecResponse
	(*StoreQueryResponse)(nil), // 8: store.StoreQueryResponse
	(*ExecStringReq)(nil),      // 9: store.ExecStringReq
	(*QueryStringReq)(nil),     // 10: store.QueryStringReq
}
var file_proto_store_proto_depIdxs = []int32{
	0,  // 0: store.Statement.params:type_name -> store.Parameter
	1,  // 1: store.Request.statements:type_name -> store.Statement
	2,  // 2: store.QueryReq.request:type_name -> store.Request
	0,  // 3: store.Values.params:type_name -> store.Parameter
	4,  // 4: store.QueryRes.values:type_name -> store.Values
	6,  // 5: store.StoreExecResponse.results:type_name -> store.ExecRes
	5,  // 6: store.StoreQueryResponse.results:type_name -> store.QueryRes
	2,  // 7: store.Store.Execute:input_type -> store.Request
	3,  // 8: store.Store.Query:input_type -> store.QueryReq
	9,  // 9: store.Store.ExecString:input_type -> store.ExecStringReq
	10, // 10: store.Store.QueryString:input_type -> store.QueryStringReq
	7,  // 11: store.Store.Execute:output_type -> store.StoreExecResponse
	8,  // 12: store.Store.Query:output_type -> store.StoreQueryResponse
	7,  // 13: store.Store.ExecString:output_type -> store.StoreExecResponse
	8,  // 14: store.Store.QueryString:output_type -> store.StoreQueryResponse
	11, // [11:15] is the sub-list for method output_type
	7,  // [7:11] is the sub-list for method input_type
	7,  // [7:7] is the sub-list for extension type_name
	7,  // [7:7] is the sub-list for extension extendee
	0,  // [0:7] is the sub-list for field type_name
}

func init() { file_proto_store_proto_init() }
func file_proto_store_proto_init() {
	if File_proto_store_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_store_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Parameter); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Statement); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Values); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryRes); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecRes); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreExecResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreQueryResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecStringReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_store_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryStringReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_proto_store_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Parameter_I)(nil),
		(*Parameter_D)(nil),
		(*Parameter_B)(nil),
		(*Parameter_Y)(nil),
		(*Parameter_S)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_store_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_store_proto_goTypes,
		DependencyIndexes: file_proto_store_proto_depIdxs,
		MessageInfos:      file_proto_store_proto_msgTypes,
	}.Build()
	File_proto_store_proto = out.File
	file_proto_store_proto_rawDesc = nil
	file_proto_store_proto_goTypes = nil
	file_proto_store_proto_depIdxs = nil
}
