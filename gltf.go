// Package gltf 实现了加载glTF格式3D模型的数据结构和方法
package gltf

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"
)

const (
	ChunkJSON = 0x4E4F534A
	ChunkBIN  = 0x004E4942
)

type ComponentType uint32

const (
	BYTE           ComponentType = 5120
	UNSIGNED_BYTE  ComponentType = 5121
	SHORT          ComponentType = 5122
	UNSIGNED_SHORT ComponentType = 5123
	UNSIGNED_INT   ComponentType = 5125
	FLOAT          ComponentType = 5126
)

func (t *ComponentType) UnmarshalJSON(b []byte) error {
	var x uint32
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	switch ComponentType(x) {
	case BYTE, UNSIGNED_BYTE, SHORT, UNSIGNED_SHORT, UNSIGNED_INT, FLOAT:
		*t = ComponentType(x)
	default:
		return fmt.Errorf("bad componentType %04d", x)
	}
	return nil
}

func (t ComponentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint32(t))
}

func (t ComponentType) NumBytes() uint32 {
	switch t {
	case BYTE, UNSIGNED_BYTE:
		return 1
	case SHORT, UNSIGNED_SHORT:
		return 2
	case UNSIGNED_INT, FLOAT:
		return 4
	default:
		return 0
	}
}

func (t ComponentType) String() string {
	switch t {
	case BYTE:
		return "BYTE"
	case UNSIGNED_BYTE:
		return "UNSIGNED_BYTE"
	case SHORT:
		return "SHORT"
	case UNSIGNED_SHORT:
		return "UNSIGNED_SHORT"
	case UNSIGNED_INT:
		return "UNSIGNED_INT"
	case FLOAT:
		return "FLOAT"
	default:
		return fmt.Sprint(uint32(t))
	}

}

type DataType uint32

const (
	SCALAR DataType = iota
	VEC2
	VEC3
	VEC4
	MAT2
	MAT3
	MAT4
)

func (a *DataType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "SCALAR":
		*a = SCALAR
	case "VEC2":
		*a = VEC2
	case "VEC3":
		*a = VEC3
	case "VEC4":
		*a = VEC4
	case "MAT2":
		*a = MAT2
	case "MAT3":
		*a = MAT3
	case "MAT4":
		*a = MAT4
	default:
		return fmt.Errorf("bad accessorType %q", s)
	}
	return nil
}

func (a DataType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + a.String() + `"`), nil
}

func (a DataType) String() string {
	switch a {
	case SCALAR:
		return "SCALAR"
	case VEC2:
		return "VEC2"
	case VEC3:
		return "VEC3"
	case VEC4:
		return "VEC4"
	case MAT2:
		return "MAT2"
	case MAT3:
		return "MAT3"
	case MAT4:
		return "MAT4"
	default:
		return ""
	}
}

func (a DataType) NumBytes() uint32 {
	switch a {
	case SCALAR:
		return 1
	case VEC2:
		return 2
	case VEC3:
		return 3
	case VEC4:
		return 4
	case MAT2:
		return 4
	case MAT3:
		return 9
	case MAT4:
		return 16
	default:
		return 0
	}
}

type (
	Translation [3]float32
	Scale       [3]float32
	Rotation    [4]float32
	Matrix      [16]float32
)

const (
	POINTS         = 0
	LINES          = 1
	LINE_LOOP      = 2
	LINE_STRIP     = 3
	TRIANGLES      = 4
	TRIANGLE_STRIP = 5
	TRIANGLE_FAN   = 6
)

const (
	NEAREST                = 9728
	LINEAR                 = 9729
	NEAREST_MIPMAP_NEAREST = 9984
	LINEAR_MIPMAP_NEAREST  = 9985
	NEAREST_MIPMAP_LINEAR  = 9986
	LINEAR_MIPMAP_LINEAR   = 9987
)

const (
	CLAMP_TO_EDGE   = 33071
	MIRRORED_REPEAT = 33648
	REPEAT          = 10497
)

func newFloat1() *float32 {
	p := new(float32)
	*p = 1
	return p
}

type Asset struct {
	Version   string
	Generator string
	Copyright string
}

type Buffer struct {
	URI        string
	ByteLength uint32
	Name       string

	data []byte
	mime string
}

func (b Buffer) IsExternal() bool {
	// glTF Buffer referring to GLB-stored BIN chunk, must have buffer.uri property undefined
	return b.URI != ""
}

type Node struct {
	Name        string
	Mesh        *uint32
	Skin        *uint32
	Children    []uint32
	Camera      *uint32
	Translation *Translation
	Scale       *Scale
	Rotation    *Rotation
	Matrix      *Matrix
}

func (n *Node) assignDefaults() {
}

type Scene struct {
	Name  string
	Nodes []uint32
}

type BufferView struct {
	Buffer     uint32
	ByteLength uint32
	ByteOffset uint32  // default 0
	ByteStride *uint32 // nil or  4 to 252
	Target     *uint32 // 34962 ARRAY_BUFFER, 34963 ELEMENT_ARRAY_BUFFER
	Name       string
}

type Accessor struct {
	BufferView    uint32
	ByteOffset    uint32
	ComponentType ComponentType
	Count         uint32
	Max           []float32
	Min           []float32
	Type          DataType

	Sparse *struct {
		Count   uint32
		Indices struct {
			BufferView    uint32
			ByteOffset    uint32
			ComponentType ComponentType
		}
		Values struct {
			BufferView uint32
			ByteOffset uint32
		}
	}
}

func (a Accessor) NumElemBytes() uint32 {
	return a.ComponentType.NumBytes() * a.Type.NumBytes()
}

func (a Accessor) NumTotalBytes() uint32 {
	return a.NumElemBytes() * a.Count
}

type Attributes struct {
	POSITION   *uint32
	NORMAL     *uint32
	TANGENT    *uint32
	TEXCOORD_0 *uint32
	TEXCOORD_1 *uint32
	TEXCOORD_2 *uint32
	TEXCOORD_3 *uint32
	COLOR_0    *uint32
	COLOR_1    *uint32
	COLOR_2    *uint32
	COLOR_3    *uint32
	JOINTS_0   *uint32
	JOINTS_1   *uint32
	JOINTS_2   *uint32
	JOINTS_3   *uint32
	WEIGHTS_0  *uint32
	WEIGHTS_1  *uint32
	WEIGHTS_2  *uint32
	WEIGHTS_3  *uint32
}

// 顶点变形动画的目标.
type MorphTarget struct {
	NORMAL   *uint32
	POSITION *uint32
	TANGENT  *uint32

	// TODO: 在glTF2.0规范里, 变形动画的目标是map而不是struct.
	//  也就是说目标是可扩展的, 规范里举了"_TEMPERATURE"的例子.
	//  考虑改成map
}

// 几何图元
type Primitive struct {
	Attributes Attributes    // 顶点属性
	Indices    *uint32       // 顶点索引
	Material   *uint32       // 材质
	Mode       *uint32       // 图元类型 default 4 TRIANGLES
	Targets    []MorphTarget // 顶点变形动画的目标
}

func (pri *Primitive) assignDefaults() {
	if pri.Mode == nil {
		pri.Mode = new(uint32)
		*pri.Mode = TRIANGLES
	}
}

type Mesh struct {
	Primitives []Primitive
	Weights    []float32
	Name       string
}

func (m *Mesh) assignDefaults() {
	for i := range m.Primitives {
		m.Primitives[i].assignDefaults()
	}
}

// 皮肤.
// glTF的皮肤可以理解为Bind Pose, 不是通常意义上的皮肤.
// 它通常和Mesh配套使用, 把Mesh绑到骨骼上. 当然它也可以不绑Mesh, Bind Pose本身可独立使用, 比如用来画骨骼.
// 皮肤里最有意义的数据是IBM数据, IBM和Bind Pose其实是一回事, 知道骨骼姿势就能算出IBM, 反之亦然.
type Skin struct {
	Name                string
	InverseBindMatrices *uint32
	Joints              []uint32

	// 骨骼的公共根节点LCA(非必填).
	// glTF文档里虽然没明确指出骨骼的范围, 我们理解为当前Skin的骨骼的LCA.
	Skeleton *uint32
}

type Texture struct {
	Name    string
	Sampler *uint32 // When undefined, a sampler with repeat wrapping and auto filtering should be used
	Source  *uint32 // When undefined, it is expected that an extension or other mechanism will supply an alternate texture source
}

type TextureInfo struct {
	Index    uint32
	TexCoord uint32
}

type Image struct {
	URI        string
	BufferView *uint32
	MimeType   string
	Name       string
}

type Sampler struct {
	MagFilter uint32
	MinFilter uint32
	WrapS     uint32
	WrapT     uint32
}

type NormalTextureInfo struct {
	Index    uint32
	TexCoord uint32
	Scale    *float32 // default 1
}

type OcclusionTextureInfo struct {
	Index    uint32
	TexCoord uint32
	Strength *float32 // default 1
}

type KHR_materials_pbrSpecularGlossiness struct {
	DiffuseFactor             *[4]float32
	SpecularFactor            *[3]float32
	DiffuseTexture            TextureInfo
	GlossinessFactor          *float32
	SpecularGlossinessTexture TextureInfo
}

type KHR_materials_clearcoat struct {
	ClearcoatFactor           float32
	ClearcoatTexture          *TextureInfo
	ClearcoatRoughnessFactor  float32
	ClearcoatRoughnessTexture *TextureInfo
	ClearcoatNormalTexture    *NormalTextureInfo
}

type KHR_materials_unlit struct {
	// 无内容
}

type Material struct {
	Name string

	PbrMetallicRoughness *struct {
		BaseColorFactor  *[4]float32 // default 1
		BaseColorTexture *TextureInfo

		MetallicFactor           *float32 // default 1
		RoughnessFactor          *float32 // default 1
		MetallicRoughnessTexture *TextureInfo
	}

	NormalTexture *NormalTextureInfo

	OcclusionTexture *OcclusionTextureInfo

	EmissiveFactor  [3]float32 // default [0 0 0]
	EmissiveTexture *TextureInfo

	AlphaMode   string
	AlphaCutoff *float32 // default 0.5
	DoubleSided bool

	Extensions struct {
		KHR_materials_pbrSpecularGlossiness *KHR_materials_pbrSpecularGlossiness
		KHR_materials_clearcoat             *KHR_materials_clearcoat
		KHR_materials_unlit                 *KHR_materials_unlit
	}
}

func (m *Material) assignDefaults() {
	if p := m.PbrMetallicRoughness; p != nil {
		if p.BaseColorFactor == nil {
			p.BaseColorFactor = &[4]float32{1, 1, 1, 1}
		}
		if p.MetallicFactor == nil {
			p.MetallicFactor = newFloat1()
		}
		if p.RoughnessFactor == nil {
			p.RoughnessFactor = newFloat1()
		}
	}

	if p := m.NormalTexture; p != nil {
		if p.Scale == nil {
			p.Scale = newFloat1()
		}
	}

	if p := m.OcclusionTexture; p != nil {
		if p.Strength == nil {
			p.Strength = newFloat1()
		}
	}

	if m.AlphaCutoff == nil {
		m.AlphaCutoff = new(float32)
		*m.AlphaCutoff = 0.5
	}

	if m.Extensions.KHR_materials_pbrSpecularGlossiness != nil {
		if m.PbrMetallicRoughness != nil && m.PbrMetallicRoughness.BaseColorTexture == nil {
			// 有的模型, 有SpecularGlossiness, 又有一个空的 PbrMetallicRoughness:{}
			// 这种PbrMetallicRoughness我们认为是无效的, 直接去掉
			m.PbrMetallicRoughness = nil
		}
		if m.Extensions.KHR_materials_pbrSpecularGlossiness.DiffuseFactor == nil {
			m.Extensions.KHR_materials_pbrSpecularGlossiness.DiffuseFactor = &[4]float32{1, 1, 1, 1}
		}
		if m.Extensions.KHR_materials_pbrSpecularGlossiness.SpecularFactor == nil {
			m.Extensions.KHR_materials_pbrSpecularGlossiness.SpecularFactor = &[3]float32{1, 1, 1}
		}
		if m.Extensions.KHR_materials_pbrSpecularGlossiness.GlossinessFactor == nil {
			m.Extensions.KHR_materials_pbrSpecularGlossiness.GlossinessFactor = newFloat1()
		}
	}
}

type Camera struct {
	Name        string
	Type        string
	Perspective *struct {
		AspectRatio float32
		Yfov        float32
		Zfar        float32
		Znear       float32
	}
	Orthographic *struct {
		Xmag  float32
		Ymag  float32
		Zfar  float32
		Znear float32
	}
}

type AnimationSampler struct {
	KeyTimes      uint32 `json:"Input"`  // glTF的Input是关键帧时间点
	KeyValues     uint32 `json:"Output"` // glTF的Output是关键帧的各参数值
	Interpolation string // "LINEAR" "STEP" "CUBICSPLINE", ""=="LINEAR"
}

type AnimationTarget struct {
	Node uint32
	Path string //    "translation"  "rotation"  "scale" "weights"
}

type AnimationChannel struct {
	AniSampler uint32 `json:"Sampler"` // 指向所属Animation的Samplers, 不是数据的Sampler
	Target     AnimationTarget
}

type Animation struct {
	Channels    []AnimationChannel
	AniSamplers []AnimationSampler `json:"Samplers"`
	Name        string
}

type GLTF struct {
	Asset       Asset
	Nodes       []*Node
	Scenes      []*Scene
	Scene       *uint32
	Accessors   []*Accessor
	Meshes      []*Mesh
	Skins       []*Skin
	Textures    []*Texture
	Images      []*Image
	Samplers    []*Sampler
	Materials   []*Material
	Cameras     []*Camera
	BufferViews []*BufferView
	Buffers     []*Buffer
	Animations  []*Animation

	ExtensionsUsed     []string
	ExtensionsRequired []string

	BIN []byte `json:"-"`
}

func (p *GLTF) assignDefaults() {
	for i := range p.Nodes {
		p.Nodes[i].assignDefaults()
	}
	for i := range p.Materials {
		p.Materials[i].assignDefaults()
	}
	for i := range p.Meshes {
		p.Meshes[i].assignDefaults()
	}
}

func mimeDataURI(uri string) (mime string) {
	if !strings.HasSuffix(uri, "data:") {
		return
	}
	mime = uri[5:] // trim "data:"
	pos := strings.IndexByte(uri, ',')
	if pos != -1 {
		mime = uri[:pos]
	}
	if strings.HasSuffix(mime, ";base64") {
		mime = strings.TrimSuffix(mime, ";base64")
	}
	return
}

func decodeDataURI(uri string) (data []byte, mime string, err error) {
	// data:[<mediatype>][;base64],<data>
	// example "data:application/octet-stream;base64,AACAvwAA....
	uri = uri[5:] // trim "data:"
	pos := strings.IndexByte(uri, ',')
	if pos == -1 {
		return nil, "", errors.New("invalid data URI")
	}
	mime = uri[:pos]
	base64Enc := strings.HasSuffix(mime, ";base64")
	if base64Enc {
		mime = strings.TrimSuffix(mime, ";base64")
		data, err = base64.StdEncoding.DecodeString(uri[pos+1:])
	} else {
		data = []byte(uri[pos+1:])
	}
	if err != nil {
		return
	}
	if mime == "" {
		if base64Enc {
			mime = `application/octet-stream`
		} else {
			mime = `text/plain;charset=US-ASCII`
		}
	}
	return
}

func (p *GLTF) ReadBuffer(i uint32) (data []byte, mime string, err error) {
	if i >= uint32(len(p.Buffers)) {
		return nil, "", fmt.Errorf("buffer %d not found", i)
	}
	b := p.Buffers[i]
	if len(b.data) != 0 {
		return b.data, b.mime, nil
	}
	defer func() {
		if err == nil {
			if len(data) > int(b.ByteLength) {
				data = data[:b.ByteLength]
			}
			b.data, b.mime = data, mime
		}
	}()
	if b.URI == "" {
		if len(p.BIN) == 0 {
			return nil, "", errors.New("no BIN chunck or empty")
		}
		return p.BIN, `application/octet-stream`, nil
	}
	if strings.HasPrefix(b.URI, "data:") {
		return decodeDataURI(b.URI)
	}

	// 其他URI, 按格式文档, 可能有非本地文件. 但目前我们只支持本地文件, 如果读不出来则失败
	data, err = os.ReadFile(b.URI)
	if err == nil {
		mime = `application/octet-stream`
	}
	return
}

func (p *GLTF) ReadBufferView(i uint32) (data []byte, mime string, bv *BufferView, err error) {
	if i >= uint32(len(p.BufferViews)) {
		return nil, "", nil, fmt.Errorf("buffer view %d not found", i)
	}
	bv = p.BufferViews[i]
	buf, mime, err := p.ReadBuffer(bv.Buffer)
	if err != nil {
		return
	}
	if bv.ByteOffset > uint32(len(buf)) || bv.ByteOffset+bv.ByteLength > uint32(len(buf)) {
		return nil, "", nil, fmt.Errorf("buffer view overflow buffer")
	}
	data = buf[bv.ByteOffset : bv.ByteOffset+bv.ByteLength]
	return
}

func (p *GLTF) ReadAccessor(i uint32) (data []byte, stride uint32, a *Accessor, err error) {
	if i >= uint32(len(p.Accessors)) {
		return nil, 0, nil, fmt.Errorf("accessor %d not found", i)
	}
	a = p.Accessors[i]
	if a.Count == 0 {
		return nil, 0, nil, fmt.Errorf("accessor.Count == 0")
	}
	if a.Sparse != nil {
		return nil, 0, nil, fmt.Errorf("unsupported sparse accessor")
	}
	buf, _, bv, err := p.ReadBufferView(a.BufferView)
	if err != nil {
		return
	}
	// if a.ByteOffset > uint32(len(buf)) {
	// 	return nil, 0, nil, fmt.Errorf("accessor overflow buffer view")
	// }
	sizeElem := a.NumElemBytes()
	if bv.ByteStride == nil {
		stride = sizeElem
	} else {
		stride = *bv.ByteStride
	}

	// Each accessor must fit its bufferView, i.e.,
	//  accessor.byteOffset + STRIDE * (accessor.count - 1) + SIZE_OF_ELEMENT
	// must be less than or equal to bufferView.length.
	accBytesLen := a.ByteOffset + stride*(a.Count-1) + sizeElem
	if accBytesLen > bv.ByteLength {
		return nil, 0, nil, fmt.Errorf("accessor overflow buffer view")
	}

	data = buf[a.ByteOffset:accBytesLen]

	return
}

func (p *GLTF) ReadImage(i uint32) (data []byte, mime string, img *Image, err error) {
	if i >= uint32(len(p.Images)) {
		return nil, "", nil, fmt.Errorf("image %d not found", i)
	}
	img = p.Images[i]

	// buffer view 优先于 URI
	if img.BufferView != nil {
		if data, mime, _, err = p.ReadBufferView(*img.BufferView); err != nil {
			return
		}
		if img.MimeType != "" {
			mime = img.MimeType
		}
		return
	}

	// URI 可以是 data URI
	if strings.HasPrefix(img.URI, "data:") {
		if data, mime, err = decodeDataURI(img.URI); err != nil {
			return
		}
		if img.MimeType != "" {
			mime = img.MimeType
		}
		return
	}

	// 其他URI, 按格式文档, 可能有非本地文件. 但目前我们只支持本地文件, 如果读不出来则失败
	if data, err = os.ReadFile(img.URI); err != nil {
		return
	}
	mime = img.MimeType
	return
}

// 把数据块解码成结构体. 如果没有二进制数据块, binChunk可以为空.
func Decode(jsonChunk, binChunk []byte) (p *GLTF, err error) {
	p = new(GLTF)
	if err = json.Unmarshal(jsonChunk, p); err != nil {
		p = nil
		return
	}
	p.assignDefaults()
	p.BIN = binChunk
	return
}

// 把GLB格式解码为JSON和BIN两个块. 考虑到3D软件都是运行在大内存的电脑里, 我们一次把所有内容都读进来.
func ReadGLB(r io.Reader) (json, bin []byte, err error) {
	// 读第一字节, 判断有没有可能是glb
	var magic [4]byte
	if _, err = r.Read(magic[:1]); err != nil {
		return
	}

	if magic[0] != 0x67 {
		// 不可能是glb, 当作裸的glTF来解码
		var tail []byte // TODO: 可以用byte.Buffer减少一次copy
		if tail, err = io.ReadAll(r); err != nil {
			return
		}
		json = make([]byte, 1, 1+len(tail))
		json[0] = magic[0]
		json = append(json, tail...)
		return
	}

	// 只能是glb, 或者错误的格式
	if _, err = io.ReadFull(r, magic[1:]); err != nil {
		return
	}

	// 0x46546C67
	if magic[1] != 0x6C || magic[2] != 0x54 || magic[3] != 0x46 {
		err = errors.New("not a glTF file")
		return
	}

	var version, length uint32
	if err = binary.Read(r, binary.LittleEndian, &version); err != nil {
		return
	}
	if version != 2 {
		err = fmt.Errorf("unsupported glTF version %d", version)
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return
	}

	var chunkType uint32
	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return
	}
	if err = binary.Read(r, binary.LittleEndian, &chunkType); err != nil {
		return
	}
	if chunkType != ChunkJSON {
		err = fmt.Errorf("first chunk is %08X, not JSON", chunkType)
		return
	}

	json = make([]byte, length)
	if _, err = io.ReadFull(r, json); err != nil {
		json = nil
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		if err == io.EOF {
			err = nil // EOF表示一个字节都读不到了, 这种仍是合法的glb文件
		}
		return
	}
	if err = binary.Read(r, binary.LittleEndian, &chunkType); err != nil {
		return
	}
	if chunkType != ChunkBIN {
		err = fmt.Errorf("second chunk is %08X, not BIN", chunkType)
		return
	}
	bin = make([]byte, length)
	if _, err = io.ReadFull(r, bin); err != nil {
		bin = nil
		return
	}

	return
}

func ReadDecodeGLB(r io.Reader) (p *GLTF, err error) {
	var json, bin []byte
	if json, bin, err = ReadGLB(r); err != nil {
		return
	}
	//os.WriteFile("D:/test.json", json, 0666)
	return Decode(json, bin)
}

// 把JSON和BIN两个块组合成GLB格式写入文件
func WriteGLB(w io.Writer, json, bin []byte) (err error) {
	magic := uint32(0x46546C67)
	version := uint32(2)

	padJSON := uint32((4 - len(json)%4) % 4)
	padBIN := uint32((4 - len(bin)%4) % 4)

	if err = binary.Write(w, binary.LittleEndian, &magic); err != nil {
		return
	}

	if err = binary.Write(w, binary.LittleEndian, &version); err != nil {
		return
	}

	length := 12 + 12 + uint32(len(json)) + padJSON + 12 + uint32(len(bin)) + padBIN
	if err = binary.Write(w, binary.LittleEndian, &length); err != nil {
		return
	}

	length = uint32(len(json)) + padJSON
	if err = binary.Write(w, binary.LittleEndian, &length); err != nil {
		return
	}
	chunkType := uint32(ChunkJSON)
	if err = binary.Write(w, binary.LittleEndian, &chunkType); err != nil {
		return
	}

	if _, err = w.Write(json); err != nil {
		return
	}

	if padJSON != 0 {
		pad := [3]byte{' ', ' ', ' '}
		if _, err = w.Write(pad[:padJSON]); err != nil {
			return
		}
	}

	length = uint32(len(bin)) + padBIN
	if length == 0 {
		return // 无BIN可以不输出
	}

	if err = binary.Write(w, binary.LittleEndian, &length); err != nil {
		return
	}
	chunkType = uint32(ChunkBIN)
	if err = binary.Write(w, binary.LittleEndian, &chunkType); err != nil {
		return
	}

	if _, err = w.Write(bin); err != nil {
		return
	}

	if padBIN != 0 {
		pad := [3]byte{0, 0, 0}
		if _, err = w.Write(pad[:padBIN]); err != nil {
			return
		}
	}
	return
}
