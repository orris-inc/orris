# 节点管理领域功能文档

## 概述

节点管理领域（Node Domain）是代理服务系统的核心模块，负责管理代理节点、节点分组、订阅生成和流量统计等功能。该领域遵循领域驱动设计（DDD）原则，与订阅系统深度集成，采用节点池共享模式，支持多种订阅格式和协议。

### 领域定位

节点管理领域作为独立的业务域，提供以下核心能力：

- **节点生命周期管理**：创建、配置、监控、删除代理节点
- **节点分组策略**：灵活的节点分组和权限控制
- **多格式订阅生成**：支持主流客户端的订阅格式
- **流量统计监控**：实时流量数据采集和统计分析
- **与订阅系统集成**：基于订阅计划的节点访问控制

### 节点池共享模式

本系统采用节点池共享架构：

```
┌─────────────────────────────────────────────────────────┐
│                     节点池 (Node Pool)                    │
│                                                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Node US-01 │  │  Node JP-01 │  │  Node HK-01 │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Node US-02 │  │  Node JP-02 │  │  Node HK-02 │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
           │                 │                 │
           ▼                 ▼                 ▼
    ┌────────────┐    ┌────────────┐    ┌────────────┐
    │ 节点组 A    │    │ 节点组 B    │    │ 节点组 C    │
    │ (美国节点)  │    │ (亚洲节点)  │    │ (全球节点)  │
    └────────────┘    └────────────┘    └────────────┘
           │                 │                 │
           ▼                 ▼                 ▼
    ┌────────────┐    ┌────────────┐    ┌────────────┐
    │ 订阅计划 1  │    │ 订阅计划 2  │    │ 订阅计划 3  │
    │ (基础版)    │    │ (专业版)    │    │ (企业版)    │
    └────────────┘    └────────────┘    └────────────┘
```

**特点：**
- 节点可被多个节点组引用
- 节点组可被多个订阅计划关联
- 订阅计划定义用户可访问的节点组
- 灵活组合，按需分配资源

### 支持的协议

当前版本支持 **Shadowsocks** 协议：

- **加密方式**: aes-256-gcm, aes-128-gcm, chacha20-ietf-poly1305
- **插件支持**: obfs (http/tls), v2ray-plugin
- **协议特点**: 轻量、高效、安全

### 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    管理端 (Admin)                         │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ 节点管理      │  │ 节点组管理    │  │ 流量统计      │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└────────────────────────┬─────────────────────────────────┘
                         │
                         │ RESTful API
                         │
┌────────────────────────▼─────────────────────────────────┐
│                    应用服务层                             │
│                                                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │         Use Cases (应用层用例)                    │   │
│  │  • CreateNodeUseCase                             │   │
│  │  • GenerateSubscriptionUseCase                   │   │
│  │  • ReportNodeDataUseCase                         │   │
│  └──────────────────────────────────────────────────┘   │
└────────────────────────┬─────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────┐
│                    领域层                                 │
│                                                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │   Node   │  │NodeGroup │  │ Traffic  │              │
│  │  聚合根   │  │  聚合根   │  │   实体   │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└────────────────────────┬─────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────┐
│                  基础设施层                               │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ PostgreSQL   │  │    Redis     │  │ 消息队列      │  │
│  │  (持久化)     │  │   (缓存)     │  │ (异步处理)    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                    代理节点                               │
│                                                           │
│  每 30 秒 HTTP POST 上报数据 (Bearer Token)               │
│  • 流量统计 (上传/下载)                                   │
│  • 在线用户数                                            │
│  • 节点状态                                              │
└────────────────────────┬─────────────────────────────────┘
                         │
                         │ HTTP API
                         │
┌────────────────────────▼─────────────────────────────────┐
│              订阅 URL 端点 (Public API)                   │
│                                                           │
│  用户通过订阅 URL 获取节点列表                            │
│  • 验证订阅令牌                                          │
│  • 过滤节点组                                            │
│  • 生成多格式配置 (Base64/Clash/V2Ray/SIP008/Surge)     │
└─────────────────────────────────────────────────────────┘
```

## 领域模型

### 聚合根

#### Node 聚合（代理节点）

Node 聚合是节点管理领域的核心，封装了代理节点的完整配置和状态管理。

**核心属性：**
- `id`: 节点唯一标识
- `name`: 节点名称（如 "US-01"）
- `serverAddress`: 服务器地址（值对象）
- `serverPort`: 服务器端口
- `encryptionConfig`: 加密配置（值对象）
- `pluginConfig`: 插件配置（值对象，可选）
- `status`: 节点状态（值对象）
- `metadata`: 节点元信息（值对象）
- `apiToken`: API Token（用于节点上报）
- `tokenHash`: Token 哈希值（SHA256）
- `maxUsers`: 最大用户数（0 表示无限制）
- `trafficLimit`: 流量限制（字节，0 表示无限制）
- `trafficUsed`: 已使用流量（字节）
- `sortOrder`: 排序顺序
- `version`: 乐观锁版本号
- `createdAt`: 创建时间
- `updatedAt`: 更新时间

**业务方法：**

```go
// 激活节点
func (n *Node) Activate() error

// 停用节点
func (n *Node) Deactivate() error

// 进入维护状态
func (n *Node) EnterMaintenance(reason string) error

// 恢复正常状态
func (n *Node) ExitMaintenance() error

// 更新服务器地址
func (n *Node) UpdateServerAddress(address ServerAddress) error

// 更新加密配置
func (n *Node) UpdateEncryption(config EncryptionConfig) error

// 更新插件配置
func (n *Node) UpdatePlugin(config *PluginConfig) error

// 生成 API Token
func (n *Node) GenerateAPIToken() (string, error)

// 验证 API Token
func (n *Node) VerifyAPIToken(plainToken string) bool

// 记录流量
func (n *Node) RecordTraffic(upload, download uint64) error

// 检查流量是否超限
func (n *Node) IsTrafficExceeded() bool

// 重置流量
func (n *Node) ResetTraffic() error

// 检查是否可用
func (n *Node) IsAvailable() bool
```

#### NodeGroup 聚合（节点组）

NodeGroup 聚合管理节点分组，支持将多个节点组织在一起，并关联到订阅计划。

**核心属性：**
- `id`: 节点组唯一标识
- `name`: 节点组名称（如 "美国节点组"）
- `description`: 节点组描述
- `nodeIDs`: 包含的节点 ID 列表
- `subscriptionPlanIDs`: 关联的订阅计划 ID 列表
- `isPublic`: 是否公开可见
- `sortOrder`: 排序顺序
- `metadata`: 元数据（JSON）
- `createdAt`: 创建时间
- `updatedAt`: 更新时间

**业务方法：**

```go
// 添加节点
func (ng *NodeGroup) AddNode(nodeID uint) error

// 移除节点
func (ng *NodeGroup) RemoveNode(nodeID uint) error

// 检查是否包含节点
func (ng *NodeGroup) ContainsNode(nodeID uint) bool

// 关联订阅计划
func (ng *NodeGroup) AssociatePlan(planID uint) error

// 取消关联订阅计划
func (ng *NodeGroup) DisassociatePlan(planID uint) error

// 检查是否关联计划
func (ng *NodeGroup) IsAssociatedWithPlan(planID uint) bool

// 获取节点数量
func (ng *NodeGroup) NodeCount() int
```

### 实体

#### NodeTraffic（节点流量统计）

NodeTraffic 实体记录节点的流量使用情况，支持按时间聚合统计。

**属性：**
- `ID`: 记录 ID
- `NodeID`: 关联节点 ID
- `UserID`: 用户 ID（如果可追踪）
- `SubscriptionID`: 订阅 ID（如果可追踪）
- `Upload`: 上传流量（字节）
- `Download`: 下载流量（字节）
- `Total`: 总流量（字节）
- `Period`: 统计周期（时间戳，按小时/天聚合）
- `CreatedAt`: 记录时间
- `UpdatedAt`: 更新时间

**业务方法：**

```go
// 累加流量
func (nt *NodeTraffic) Accumulate(upload, download uint64)

// 获取总流量
func (nt *NodeTraffic) TotalTraffic() uint64

// 获取流量比率（上传/总流量）
func (nt *NodeTraffic) UploadRatio() float64
```

#### NodeAccessLog（节点访问日志）

NodeAccessLog 实体记录用户访问节点的详细日志，用于审计和分析。

**属性：**
- `ID`: 日志 ID
- `NodeID`: 节点 ID
- `UserID`: 用户 ID
- `SubscriptionID`: 订阅 ID
- `SubscriptionToken`: 订阅令牌（脱敏）
- `ClientIP`: 客户端 IP 地址
- `UserAgent`: 用户代理
- `ConnectTime`: 连接时间
- `DisconnectTime`: 断开时间（可选）
- `Duration`: 连接时长（秒）
- `Upload`: 上传流量（字节）
- `Download`: 下载流量（字节）
- `CreatedAt`: 记录时间

### 值对象

#### ServerAddress（服务器地址）

```go
type ServerAddress struct {
    value string // IP 地址或域名
}

func NewServerAddress(address string) (ServerAddress, error) {
    // 验证 IP 地址或域名格式
    if !isValidIP(address) && !isValidDomain(address) {
        return ServerAddress{}, fmt.Errorf("invalid server address: %s", address)
    }
    return ServerAddress{value: address}, nil
}

func (sa ServerAddress) Value() string {
    return sa.value
}

func (sa ServerAddress) IsIP() bool {
    return isValidIP(sa.value)
}

func (sa ServerAddress) IsDomain() bool {
    return isValidDomain(sa.value)
}
```

#### EncryptionConfig（加密配置）

```go
type EncryptionConfig struct {
    method   string // 加密方式
    password string // 密码
}

const (
    MethodAES256GCM           = "aes-256-gcm"
    MethodAES128GCM           = "aes-128-gcm"
    MethodChacha20IETFPoly1305 = "chacha20-ietf-poly1305"
)

func NewEncryptionConfig(method, password string) (EncryptionConfig, error) {
    // 验证加密方式
    if !isValidMethod(method) {
        return EncryptionConfig{}, fmt.Errorf("unsupported encryption method: %s", method)
    }

    // 验证密码强度
    if len(password) < 8 {
        return EncryptionConfig{}, fmt.Errorf("password too short")
    }

    return EncryptionConfig{
        method:   method,
        password: password,
    }, nil
}

func (ec EncryptionConfig) Method() string {
    return ec.method
}

func (ec EncryptionConfig) Password() string {
    return ec.password
}

func (ec EncryptionConfig) ToShadowsocksURI() string {
    // method:password
    auth := fmt.Sprintf("%s:%s", ec.method, ec.password)
    return base64.URLEncoding.EncodeToString([]byte(auth))
}
```

#### PluginConfig（插件配置）

```go
type PluginConfig struct {
    plugin string            // 插件名称 (obfs, v2ray-plugin)
    opts   map[string]string // 插件选项
}

func NewObfsPlugin(mode string) PluginConfig {
    return PluginConfig{
        plugin: "obfs-local",
        opts: map[string]string{
            "obfs": mode, // http, tls
        },
    }
}

func NewV2RayPlugin(mode string, host string) PluginConfig {
    return PluginConfig{
        plugin: "v2ray-plugin",
        opts: map[string]string{
            "mode": mode, // websocket, quic
            "host": host,
        },
    }
}

func (pc PluginConfig) Plugin() string {
    return pc.plugin
}

func (pc PluginConfig) Opts() map[string]string {
    return pc.opts
}

func (pc PluginConfig) ToPluginOpts() string {
    // 转换为插件选项字符串，如: obfs=http;obfs-host=www.bing.com
    var parts []string
    for k, v := range pc.opts {
        parts = append(parts, fmt.Sprintf("%s=%s", k, v))
    }
    return strings.Join(parts, ";")
}
```

#### NodeStatus（节点状态）

```go
type NodeStatus string

const (
    NodeStatusActive      NodeStatus = "active"      // 激活
    NodeStatusInactive    NodeStatus = "inactive"    // 停用
    NodeStatusMaintenance NodeStatus = "maintenance" // 维护中
)

var NodeStatusTransitions = map[NodeStatus][]NodeStatus{
    NodeStatusInactive: {
        NodeStatusActive,
    },
    NodeStatusActive: {
        NodeStatusInactive,
        NodeStatusMaintenance,
    },
    NodeStatusMaintenance: {
        NodeStatusActive,
        NodeStatusInactive,
    },
}

func (ns NodeStatus) IsActive() bool {
    return ns == NodeStatusActive
}

func (ns NodeStatus) CanTransitionTo(target NodeStatus) bool {
    allowedTransitions, ok := NodeStatusTransitions[ns]
    if !ok {
        return false
    }

    for _, allowed := range allowedTransitions {
        if allowed == target {
            return true
        }
    }
    return false
}
```

#### NodeMetadata（节点元信息）

```go
type NodeMetadata struct {
    country     string   // 国家代码 (US, JP, HK)
    region      string   // 地区 (California, Tokyo, Hong Kong)
    tags        []string // 标签 (premium, gaming, streaming)
    description string   // 描述
}

func NewNodeMetadata(country, region string, tags []string, description string) NodeMetadata {
    return NodeMetadata{
        country:     strings.ToUpper(country),
        region:      region,
        tags:        tags,
        description: description,
    }
}

func (nm NodeMetadata) Country() string {
    return nm.country
}

func (nm NodeMetadata) Region() string {
    return nm.region
}

func (nm NodeMetadata) Tags() []string {
    return nm.tags
}

func (nm NodeMetadata) HasTag(tag string) bool {
    for _, t := range nm.tags {
        if t == tag {
            return true
        }
    }
    return false
}

func (nm NodeMetadata) DisplayName() string {
    // 格式: 国家 - 地区
    return fmt.Sprintf("%s - %s", nm.country, nm.region)
}
```

#### NodeToken（节点 Token）

```go
type NodeToken struct {
    tokenHash string    // Token 哈希值
    expiresAt *time.Time // 过期时间（nil 表示永不过期）
}

func GenerateNodeToken() (plainToken string, tokenHash string, err error) {
    // 生成 32 字节随机 token
    tokenBytes := make([]byte, 32)
    _, err = rand.Read(tokenBytes)
    if err != nil {
        return "", "", err
    }

    plainToken = base64.URLEncoding.EncodeToString(tokenBytes)

    // 计算 SHA256 哈希
    hash := sha256.Sum256([]byte(plainToken))
    tokenHash = hex.EncodeToString(hash[:])

    return plainToken, tokenHash, nil
}

func (nt NodeToken) Verify(plainToken string) bool {
    hash := sha256.Sum256([]byte(plainToken))
    tokenHash := hex.EncodeToString(hash[:])
    return subtle.ConstantTimeCompare([]byte(nt.tokenHash), []byte(tokenHash)) == 1
}

func (nt NodeToken) IsExpired() bool {
    if nt.expiresAt == nil {
        return false
    }
    return time.Now().After(*nt.expiresAt)
}
```

## 核心功能

### 1. 节点管理

#### 创建节点

**用例：** `CreateNodeUseCase`

**流程：**
1. 验证服务器地址和端口
2. 验证加密配置
3. 验证插件配置（如果有）
4. 生成 API Token
5. 创建节点聚合
6. 持久化节点
7. 触发节点创建事件

**输入：**
```go
type CreateNodeCommand struct {
    Name          string
    ServerAddress string
    ServerPort    uint16
    Method        string
    Password      string
    Plugin        *PluginConfigDTO
    Country       string
    Region        string
    Tags          []string
    Description   string
    MaxUsers      uint
    TrafficLimit  uint64
    SortOrder     int
}

type PluginConfigDTO struct {
    Plugin string
    Opts   map[string]string
}
```

**输出：**
```go
type CreateNodeResult struct {
    Node      *Node
    APIToken  string // 明文 Token（仅返回一次）
    TokenHash string
}
```

#### 更新节点

**用例：** `UpdateNodeUseCase`

**可更新内容：**
- 节点名称
- 服务器地址和端口
- 加密配置
- 插件配置
- 元信息（地区、标签、描述）
- 流量限制
- 排序顺序

**注意事项：**
- 更新加密配置会影响用户连接，需通知用户更新订阅
- 更新服务器地址需要节点程序重启

#### 删除节点

**用例：** `DeleteNodeUseCase`

**流程：**
1. 验证权限
2. 检查节点是否被节点组引用
3. 如果被引用，从所有节点组中移除
4. 撤销节点 Token
5. 删除节点数据（软删除或硬删除）
6. 触发节点删除事件

#### 生成节点 Token

**用例：** `GenerateNodeTokenUseCase`

**流程：**
1. 验证节点存在
2. 生成 32 字节随机 Token
3. 计算 SHA256 哈希
4. 存储哈希值到节点
5. 触发 Token 生成事件
6. 返回明文 Token（仅此一次）

**Token 格式：**
```
node_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

**安全特性：**
- 使用 crypto/rand 生成随机 Token
- SHA256 哈希存储
- 恒定时间比较防止时序攻击

### 2. 节点组管理

#### 创建节点组

**用例：** `CreateNodeGroupUseCase`

**流程：**
1. 验证节点组名称唯一性
2. 创建节点组聚合
3. 持久化节点组
4. 触发节点组创建事件

**输入：**
```go
type CreateNodeGroupCommand struct {
    Name        string
    Description string
    IsPublic    bool
    SortOrder   int
}
```

#### 添加节点到组

**用例：** `AddNodeToGroupUseCase`

**流程：**
1. 验证节点存在
2. 验证节点组存在
3. 检查节点是否已在组中
4. 添加节点到节点组
5. 更新节点组
6. 触发节点组更新事件

#### 关联订阅计划

**用例：** `AssociateGroupWithPlanUseCase`

**流程：**
1. 验证节点组存在
2. 验证订阅计划存在
3. 关联节点组和订阅计划
4. 触发关联事件

**关联模型：**
```go
// 多对多关系
type NodeGroupPlan struct {
    NodeGroupID        uint
    SubscriptionPlanID uint
    CreatedAt          time.Time
}
```

### 3. 订阅生成

订阅生成是节点管理领域的核心功能，支持多种主流客户端格式。

#### 生成订阅

**用例：** `GenerateSubscriptionUseCase`

**流程：**
1. 验证订阅令牌（来自 Subscription Domain）
2. 获取用户订阅信息
3. 获取订阅计划关联的节点组
4. 获取节点组中的所有节点
5. 过滤激活状态的节点
6. 根据请求格式生成配置
7. 签名配置（防篡改）
8. 记录订阅生成事件
9. 返回配置

**输入：**
```go
type GenerateSubscriptionCommand struct {
    SubscriptionToken string // 订阅令牌
    Format            string // base64, clash, v2ray, sip008, surge
    UserAgent         string // 客户端信息
}
```

**输出：**
```go
type GenerateSubscriptionResult struct {
    Format  string
    Content string // 或 []byte
    Nodes   int    // 节点数量
}
```

### 4. 订阅格式规范

#### 4.1 Base64 格式

Base64 是最基础的 Shadowsocks 订阅格式，使用换行分隔的 ss:// 链接列表，整体进行 Base64 编码。

**单个节点格式：**
```
ss://[base64(method:password)]@server:port#name
```

**示例：**
```go
func (s *SubscriptionService) GenerateBase64(nodes []*Node) string {
    var links []string

    for _, node := range nodes {
        // 1. 构造 method:password
        auth := fmt.Sprintf("%s:%s",
            node.EncryptionConfig.Method(),
            node.EncryptionConfig.Password())

        // 2. Base64 编码认证信息
        authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))

        // 3. 构造 ss:// 链接
        link := fmt.Sprintf("ss://%s@%s:%d",
            authEncoded,
            node.ServerAddress.Value(),
            node.ServerPort)

        // 4. 添加节点名称（URL 编码）
        if node.Name != "" {
            link += "#" + url.QueryEscape(node.Name)
        }

        // 5. 处理插件（如果有）
        if node.PluginConfig != nil {
            plugin := node.PluginConfig.Plugin()
            pluginOpts := node.PluginConfig.ToPluginOpts()
            link += fmt.Sprintf("?plugin=%s;%s",
                url.QueryEscape(plugin),
                url.QueryEscape(pluginOpts))
        }

        links = append(links, link)
    }

    // 6. 用换行符连接所有链接
    content := strings.Join(links, "\n")

    // 7. 整体 Base64 编码
    return base64.StdEncoding.EncodeToString([]byte(content))
}
```

**完整示例：**
```
ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM=@server1.example.com:8388#US-01
ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM=@server2.example.com:8388#JP-01
```

经过 Base64 编码后返回给客户端。

#### 4.2 Clash 格式

Clash 使用 YAML 格式配置文件。

**配置结构：**
```yaml
proxies:
  - name: "节点名称"
    type: ss
    server: server_address
    port: port
    cipher: method
    password: password
    udp: true
    plugin: plugin_name
    plugin-opts:
      key: value
```

**代码实现：**
```go
func (s *SubscriptionService) GenerateClash(nodes []*Node) string {
    type ClashProxy struct {
        Name       string            `yaml:"name"`
        Type       string            `yaml:"type"`
        Server     string            `yaml:"server"`
        Port       uint16            `yaml:"port"`
        Cipher     string            `yaml:"cipher"`
        Password   string            `yaml:"password"`
        UDP        bool              `yaml:"udp"`
        Plugin     string            `yaml:"plugin,omitempty"`
        PluginOpts map[string]string `yaml:"plugin-opts,omitempty"`
    }

    type ClashConfig struct {
        Proxies []ClashProxy `yaml:"proxies"`
    }

    config := ClashConfig{}

    for _, node := range nodes {
        proxy := ClashProxy{
            Name:     node.Name,
            Type:     "ss",
            Server:   node.ServerAddress.Value(),
            Port:     node.ServerPort,
            Cipher:   node.EncryptionConfig.Method(),
            Password: node.EncryptionConfig.Password(),
            UDP:      true,
        }

        // 添加插件配置
        if node.PluginConfig != nil {
            proxy.Plugin = node.PluginConfig.Plugin()
            proxy.PluginOpts = node.PluginConfig.Opts()
        }

        config.Proxies = append(config.Proxies, proxy)
    }

    // 序列化为 YAML
    yamlBytes, _ := yaml.Marshal(config)
    return string(yamlBytes)
}
```

**完整配置示例：**
```yaml
proxies:
  - name: "US-01"
    type: ss
    server: server1.example.com
    port: 8388
    cipher: aes-256-gcm
    password: password123
    udp: true
    plugin: obfs-local
    plugin-opts:
      obfs: http
      obfs-host: www.bing.com

  - name: "JP-01"
    type: ss
    server: server2.example.com
    port: 8388
    cipher: chacha20-ietf-poly1305
    password: password123
    udp: true
```

#### 4.3 V2Ray/V2RayN 格式

V2RayN 使用 JSON 格式，但仍使用 Shadowsocks 协议。

**配置结构：**
```json
{
  "remarks": "节点名称",
  "server": "server_address",
  "server_port": port,
  "password": "password",
  "method": "method",
  "plugin": "plugin_name",
  "plugin_opts": "options"
}
```

**代码实现：**
```go
func (s *SubscriptionService) GenerateV2Ray(nodes []*Node) string {
    type V2RayNode struct {
        Remarks    string `json:"remarks"`
        Server     string `json:"server"`
        ServerPort uint16 `json:"server_port"`
        Password   string `json:"password"`
        Method     string `json:"method"`
        Plugin     string `json:"plugin,omitempty"`
        PluginOpts string `json:"plugin_opts,omitempty"`
    }

    var v2rayNodes []V2RayNode

    for _, node := range nodes {
        v2rayNode := V2RayNode{
            Remarks:    node.Name,
            Server:     node.ServerAddress.Value(),
            ServerPort: node.ServerPort,
            Password:   node.EncryptionConfig.Password(),
            Method:     node.EncryptionConfig.Method(),
        }

        if node.PluginConfig != nil {
            v2rayNode.Plugin = node.PluginConfig.Plugin()
            v2rayNode.PluginOpts = node.PluginConfig.ToPluginOpts()
        }

        v2rayNodes = append(v2rayNodes, v2rayNode)
    }

    jsonBytes, _ := json.MarshalIndent(v2rayNodes, "", "  ")
    return string(jsonBytes)
}
```

#### 4.4 SIP008 格式

SIP008 是 Shadowsocks 的标准 JSON 配置格式。

**配置结构：**
```json
{
  "version": 1,
  "servers": [
    {
      "id": "node_id",
      "remarks": "节点名称",
      "server": "server_address",
      "server_port": port,
      "password": "password",
      "method": "method",
      "plugin": "plugin_name",
      "plugin_opts": "options"
    }
  ]
}
```

**代码实现：**
```go
func (s *SubscriptionService) GenerateSIP008(nodes []*Node) string {
    type SIP008Server struct {
        ID         string `json:"id"`
        Remarks    string `json:"remarks"`
        Server     string `json:"server"`
        ServerPort uint16 `json:"server_port"`
        Password   string `json:"password"`
        Method     string `json:"method"`
        Plugin     string `json:"plugin,omitempty"`
        PluginOpts string `json:"plugin_opts,omitempty"`
    }

    type SIP008Config struct {
        Version int            `json:"version"`
        Servers []SIP008Server `json:"servers"`
    }

    config := SIP008Config{
        Version: 1,
        Servers: []SIP008Server{},
    }

    for _, node := range nodes {
        server := SIP008Server{
            ID:         fmt.Sprintf("node_%d", node.ID),
            Remarks:    node.Name,
            Server:     node.ServerAddress.Value(),
            ServerPort: node.ServerPort,
            Password:   node.EncryptionConfig.Password(),
            Method:     node.EncryptionConfig.Method(),
        }

        if node.PluginConfig != nil {
            server.Plugin = node.PluginConfig.Plugin()
            server.PluginOpts = node.PluginConfig.ToPluginOpts()
        }

        config.Servers = append(config.Servers, server)
    }

    jsonBytes, _ := json.MarshalIndent(config, "", "  ")
    return string(jsonBytes)
}
```

#### 4.5 Surge 格式

Surge 使用类似 INI 的配置格式。

**配置结构：**
```
[Proxy]
NodeName = ss, server, port, encrypt-method=method, password=password, udp-relay=true
```

**代码实现：**
```go
func (s *SubscriptionService) GenerateSurge(nodes []*Node) string {
    var lines []string
    lines = append(lines, "[Proxy]")

    for _, node := range nodes {
        // 节点名称不能有空格，替换为下划线
        nodeName := strings.ReplaceAll(node.Name, " ", "_")

        line := fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s, udp-relay=true",
            nodeName,
            node.ServerAddress.Value(),
            node.ServerPort,
            node.EncryptionConfig.Method(),
            node.EncryptionConfig.Password())

        // 添加插件配置
        if node.PluginConfig != nil {
            if node.PluginConfig.Plugin() == "obfs-local" {
                opts := node.PluginConfig.Opts()
                if obfsMode, ok := opts["obfs"]; ok {
                    line += fmt.Sprintf(", obfs=%s", obfsMode)
                    if obfsHost, ok := opts["obfs-host"]; ok {
                        line += fmt.Sprintf(", obfs-host=%s", obfsHost)
                    }
                }
            }
        }

        lines = append(lines, line)
    }

    return strings.Join(lines, "\n")
}
```

**完整示例：**
```
[Proxy]
US_01 = ss, server1.example.com, 8388, encrypt-method=aes-256-gcm, password=password123, udp-relay=true, obfs=http, obfs-host=www.bing.com
JP_01 = ss, server2.example.com, 8388, encrypt-method=chacha20-ietf-poly1305, password=password123, udp-relay=true
```

### 5. 节点数据上报

节点程序定期向服务端上报流量和状态数据。

#### 上报接口设计

**端点：**
```
POST /nodes/report
```

**认证：**
```
Authorization: Bearer {node_token}
```

**请求体：**
```json
{
  "node_id": 123,
  "timestamp": "2024-01-20T10:30:00Z",
  "traffic": {
    "upload": 1024000,
    "download": 2048000
  },
  "online_users": 15,
  "status": "active",
  "system_info": {
    "load": 0.45,
    "memory_usage": 60.5,
    "disk_usage": 30.2
  }
}
```

**响应：**
```json
{
  "success": true,
  "config_version": "v1.2.3",
  "should_reload": false,
  "message": "data received"
}
```

#### 上报处理流程

```go
type ReportNodeDataCommand struct {
    NodeID       uint
    Timestamp    time.Time
    Upload       uint64
    Download     uint64
    OnlineUsers  int
    Status       string
    SystemLoad   float64
    MemoryUsage  float64
    DiskUsage    float64
}

type ReportNodeDataUseCase struct {
    nodeRepo    NodeRepository
    trafficRepo NodeTrafficRepository
    queue       MessageQueue
    logger      Logger
}

func (uc *ReportNodeDataUseCase) Execute(ctx context.Context, cmd ReportNodeDataCommand) error {
    // 1. 异步处理，立即返回
    uc.queue.Publish("node.report", cmd)

    return nil
}

// 异步处理器
func (uc *ReportNodeDataUseCase) HandleReport(cmd ReportNodeDataCommand) error {
    ctx := context.Background()

    // 1. 获取节点
    node, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
    if err != nil {
        uc.logger.Error("node not found", "node_id", cmd.NodeID, "error", err)
        return err
    }

    // 2. 记录流量
    err = node.RecordTraffic(cmd.Upload, cmd.Download)
    if err != nil {
        uc.logger.Error("failed to record traffic", "error", err)
        return err
    }

    // 3. 更新节点状态
    if cmd.Status != string(node.Status) {
        // 状态变更，触发事件
        oldStatus := node.Status
        // 更新状态逻辑...

        uc.logger.Info("node status changed",
            "node_id", node.ID,
            "old_status", oldStatus,
            "new_status", cmd.Status)
    }

    // 4. 持久化节点
    err = uc.nodeRepo.Update(ctx, node)
    if err != nil {
        uc.logger.Error("failed to update node", "error", err)
        return err
    }

    // 5. 记录流量统计
    traffic := &NodeTraffic{
        NodeID:   cmd.NodeID,
        Upload:   cmd.Upload,
        Download: cmd.Download,
        Period:   time.Now().Truncate(time.Hour), // 按小时聚合
    }

    err = uc.trafficRepo.RecordTraffic(ctx, traffic)
    if err != nil {
        uc.logger.Error("failed to record traffic stats", "error", err)
    }

    // 6. 检查流量超限
    if node.IsTrafficExceeded() {
        uc.logger.Warn("node traffic exceeded", "node_id", node.ID)
        // 触发告警事件
    }

    uc.logger.Info("node report processed",
        "node_id", cmd.NodeID,
        "upload", cmd.Upload,
        "download", cmd.Download,
        "online_users", cmd.OnlineUsers)

    return nil
}
```

#### Token 验证中间件

```go
func NodeTokenMiddleware(nodeRepo NodeRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取 Bearer Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        if token == authHeader {
            c.JSON(401, gin.H{"error": "invalid authorization format"})
            c.Abort()
            return
        }

        // 2. 验证 Token
        node, err := validateNodeToken(c.Request.Context(), nodeRepo, token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // 3. 检查节点状态
        if !node.IsAvailable() {
            c.JSON(403, gin.H{"error": "node unavailable"})
            c.Abort()
            return
        }

        // 4. 注入上下文
        c.Set("node", node)
        c.Set("node_id", node.ID)

        c.Next()
    }
}

func validateNodeToken(ctx context.Context, repo NodeRepository, plainToken string) (*Node, error) {
    // 计算 Token 哈希
    hash := sha256.Sum256([]byte(plainToken))
    tokenHash := hex.EncodeToString(hash[:])

    // 查询节点
    node, err := repo.GetByToken(ctx, tokenHash)
    if err != nil {
        return nil, err
    }

    // 验证 Token
    if !node.VerifyAPIToken(plainToken) {
        return nil, fmt.Errorf("token verification failed")
    }

    return node, nil
}
```

#### 节点上报示例（Python）

```python
import requests
import json
import time
import psutil

class NodeReporter:
    def __init__(self, node_id, api_url, token):
        self.node_id = node_id
        self.api_url = api_url
        self.token = token
        self.upload_total = 0
        self.download_total = 0

    def get_traffic_stats(self):
        # 从 shadowsocks 获取流量统计
        # 这里简化为获取系统网络流量
        net_io = psutil.net_io_counters()

        upload_delta = net_io.bytes_sent - self.upload_total
        download_delta = net_io.bytes_recv - self.download_total

        self.upload_total = net_io.bytes_sent
        self.download_total = net_io.bytes_recv

        return upload_delta, download_delta

    def get_online_users(self):
        # 从 shadowsocks 获取在线用户数
        # 简化实现
        return 0

    def report(self):
        upload, download = self.get_traffic_stats()
        online_users = self.get_online_users()

        payload = {
            "node_id": self.node_id,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "traffic": {
                "upload": upload,
                "download": download
            },
            "online_users": online_users,
            "status": "active",
            "system_info": {
                "load": psutil.getloadavg()[0],
                "memory_usage": psutil.virtual_memory().percent,
                "disk_usage": psutil.disk_usage('/').percent
            }
        }

        headers = {
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json"
        }

        try:
            response = requests.post(
                f"{self.api_url}/nodes/report",
                json=payload,
                headers=headers,
                timeout=10
            )

            if response.status_code == 200:
                data = response.json()
                print(f"Report success: {data}")

                # 检查是否需要重载配置
                if data.get("should_reload"):
                    print("Config changed, reloading...")
                    # 重载配置逻辑
            else:
                print(f"Report failed: {response.status_code} {response.text}")

        except Exception as e:
            print(f"Report error: {e}")

    def run(self, interval=30):
        """每 30 秒上报一次"""
        while True:
            self.report()
            time.sleep(interval)

if __name__ == "__main__":
    reporter = NodeReporter(
        node_id=123,
        api_url="https://api.example.com",
        token="node_xxxxxxxxxxxxxxxxxxxxx"
    )
    reporter.run(interval=30)
```

### 6. 流量统计

#### 记录流量

**用例：** `RecordNodeTrafficUseCase`

流量数据采用多级聚合策略：

1. **实时累计**：更新节点的 `trafficUsed` 字段
2. **按小时聚合**：写入 `node_traffic` 表，按小时统计
3. **按天聚合**：定时任务汇总每日流量
4. **按月聚合**：定时任务汇总每月流量

```go
func (uc *RecordNodeTrafficUseCase) Execute(ctx context.Context, nodeID uint, upload, download uint64) error {
    // 1. 更新节点实时流量
    node, err := uc.nodeRepo.GetByID(ctx, nodeID)
    if err != nil {
        return err
    }

    err = node.RecordTraffic(upload, download)
    if err != nil {
        return err
    }

    err = uc.nodeRepo.Update(ctx, node)
    if err != nil {
        return err
    }

    // 2. 记录按小时聚合的流量
    period := time.Now().Truncate(time.Hour)

    traffic, err := uc.trafficRepo.GetNodeTraffic(ctx, nodeID, period)
    if err != nil {
        // 创建新记录
        traffic = &NodeTraffic{
            NodeID:   nodeID,
            Period:   period,
            Upload:   upload,
            Download: download,
        }
        return uc.trafficRepo.RecordTraffic(ctx, traffic)
    }

    // 累加流量
    traffic.Accumulate(upload, download)
    return uc.trafficRepo.Update(ctx, traffic)
}
```

#### 查询流量统计

**用例：** `GetNodeTrafficStatsUseCase`

```go
type TrafficStatsQuery struct {
    NodeID    *uint
    UserID    *uint
    StartTime time.Time
    EndTime   time.Time
    Granularity string // hour, day, month
}

type TrafficStatsResult struct {
    NodeID   uint
    Period   time.Time
    Upload   uint64
    Download uint64
    Total    uint64
}

func (uc *GetNodeTrafficStatsUseCase) Execute(ctx context.Context, query TrafficStatsQuery) ([]*TrafficStatsResult, error) {
    return uc.trafficRepo.GetTrafficStats(ctx, query)
}
```

## 领域事件

### 节点事件

#### NodeCreatedEvent

```go
type NodeCreatedEvent struct {
    NodeID        uint
    Name          string
    ServerAddress string
    ServerPort    uint16
    Status        string
    CreatedBy     uint
    Timestamp     time.Time
}
```

触发时机：创建节点时

事件处理：
- 发送通知给管理员
- 记录审计日志
- 初始化监控指标

#### NodeUpdatedEvent

```go
type NodeUpdatedEvent struct {
    NodeID        uint
    UpdatedFields []string
    OldValues     map[string]interface{}
    NewValues     map[string]interface{}
    UpdatedBy     uint
    Timestamp     time.Time
}
```

触发时机：更新节点配置时

事件处理：
- 通知订阅用户更新订阅
- 记录变更历史
- 触发配置同步

#### NodeDeletedEvent

```go
type NodeDeletedEvent struct {
    NodeID    uint
    Name      string
    DeletedBy uint
    Timestamp time.Time
}
```

触发时机：删除节点时

事件处理：
- 从所有节点组移除
- 撤销节点 Token
- 清理流量统计数据
- 通知相关用户

#### NodeStatusChangedEvent

```go
type NodeStatusChangedEvent struct {
    NodeID    uint
    OldStatus string
    NewStatus string
    Reason    string
    Timestamp time.Time
}
```

触发时机：节点状态变更时

事件处理：
- 通知订阅用户
- 更新节点可用性
- 记录状态历史

### 节点组事件

#### NodeGroupCreatedEvent

```go
type NodeGroupCreatedEvent struct {
    GroupID   uint
    Name      string
    CreatedBy uint
    Timestamp time.Time
}
```

#### NodeGroupUpdatedEvent

```go
type NodeGroupUpdatedEvent struct {
    GroupID       uint
    UpdateType    string // add_node, remove_node, update_info
    NodeIDs       []uint
    UpdatedBy     uint
    Timestamp     time.Time
}
```

### 订阅事件

#### SubscriptionGeneratedEvent

```go
type SubscriptionGeneratedEvent struct {
    SubscriptionID uint
    UserID         uint
    Format         string
    NodeCount      int
    ClientIP       string
    UserAgent      string
    Timestamp      time.Time
}
```

触发时机：生成订阅时

事件处理：
- 记录访问日志
- 统计订阅访问频率
- 检测异常访问

### 流量事件

#### NodeTrafficReportedEvent

```go
type NodeTrafficReportedEvent struct {
    NodeID    uint
    Upload    uint64
    Download  uint64
    Timestamp time.Time
}
```

#### NodeTrafficExceededEvent

```go
type NodeTrafficExceededEvent struct {
    NodeID       uint
    TrafficLimit uint64
    TrafficUsed  uint64
    Timestamp    time.Time
}
```

触发时机：节点流量超限时

事件处理：
- 发送告警通知
- 自动停用节点（可选）
- 记录超限日志

## 仓储接口

### NodeRepository

```go
type NodeRepository interface {
    // 基础操作
    Create(ctx context.Context, node *Node) error
    GetByID(ctx context.Context, id uint) (*Node, error)
    GetByToken(ctx context.Context, tokenHash string) (*Node, error)
    Update(ctx context.Context, node *Node) error
    Delete(ctx context.Context, id uint) error

    // 查询
    List(ctx context.Context, filter NodeFilter) ([]*Node, int64, error)
    GetByGroupID(ctx context.Context, groupID uint) ([]*Node, error)
    GetByStatus(ctx context.Context, status NodeStatus) ([]*Node, error)
    GetAvailableNodes(ctx context.Context) ([]*Node, error)

    // 验证
    ExistsByName(ctx context.Context, name string) (bool, error)
    ExistsByAddress(ctx context.Context, address string, port uint16) (bool, error)
}

type NodeFilter struct {
    GroupID   *uint
    Status    *NodeStatus
    Country   *string
    Tags      []string
    Search    string // 搜索名称或地址
    Page      int
    PageSize  int
    SortBy    string // name, created_at, sort_order
    SortDesc  bool
}
```

### NodeGroupRepository

```go
type NodeGroupRepository interface {
    // 基础操作
    Create(ctx context.Context, group *NodeGroup) error
    GetByID(ctx context.Context, id uint) (*NodeGroup, error)
    Update(ctx context.Context, group *NodeGroup) error
    Delete(ctx context.Context, id uint) error

    // 查询
    List(ctx context.Context, filter NodeGroupFilter) ([]*NodeGroup, int64, error)
    GetByPlanID(ctx context.Context, planID uint) ([]*NodeGroup, error)
    GetPublicGroups(ctx context.Context) ([]*NodeGroup, error)

    // 节点关联
    AddNode(ctx context.Context, groupID, nodeID uint) error
    RemoveNode(ctx context.Context, groupID, nodeID uint) error
    GetNodeIDs(ctx context.Context, groupID uint) ([]uint, error)

    // 订阅计划关联
    AssociatePlan(ctx context.Context, groupID, planID uint) error
    DisassociatePlan(ctx context.Context, groupID, planID uint) error

    // 验证
    ExistsByName(ctx context.Context, name string) (bool, error)
}

type NodeGroupFilter struct {
    PlanID    *uint
    IsPublic  *bool
    Page      int
    PageSize  int
    SortBy    string
    SortDesc  bool
}
```

### NodeTrafficRepository

```go
type NodeTrafficRepository interface {
    // 记录流量
    RecordTraffic(ctx context.Context, traffic *NodeTraffic) error
    Update(ctx context.Context, traffic *NodeTraffic) error

    // 查询
    GetNodeTraffic(ctx context.Context, nodeID uint, period time.Time) (*NodeTraffic, error)
    GetUserTraffic(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*NodeTraffic, error)
    GetTrafficStats(ctx context.Context, query TrafficStatsQuery) ([]*TrafficStatsResult, error)

    // 聚合统计
    AggregateDaily(ctx context.Context, date time.Time) error
    AggregateMonthly(ctx context.Context, month time.Time) error

    // 清理
    DeleteOldRecords(ctx context.Context, before time.Time) error
}

type TrafficStatsQuery struct {
    NodeID      *uint
    UserID      *uint
    StartTime   time.Time
    EndTime     time.Time
    Granularity string // hour, day, month
}
```

### NodeAccessLogRepository

```go
type NodeAccessLogRepository interface {
    // 创建日志
    Create(log *NodeAccessLog) error
    BatchCreate(logs []*NodeAccessLog) error

    // 查询
    List(ctx context.Context, filter AccessLogFilter) ([]*NodeAccessLog, int64, error)
    GetByNodeID(ctx context.Context, nodeID uint, limit int) ([]*NodeAccessLog, error)
    GetByUserID(ctx context.Context, userID uint, limit int) ([]*NodeAccessLog, error)

    // 统计
    CountByNode(ctx context.Context, nodeID uint, startTime, endTime time.Time) (int64, error)
    CountByUser(ctx context.Context, userID uint, startTime, endTime time.Time) (int64, error)

    // 清理
    DeleteOldLogs(ctx context.Context, before time.Time) error
}

type AccessLogFilter struct {
    NodeID         *uint
    UserID         *uint
    SubscriptionID *uint
    ClientIP       *string
    StartTime      *time.Time
    EndTime        *time.Time
    Page           int
    PageSize       int
}
```

## 应用层用例

### 节点管理用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| CreateNodeUseCase | 创建节点 | CreateNodeCommand |
| UpdateNodeUseCase | 更新节点 | UpdateNodeCommand |
| DeleteNodeUseCase | 删除节点 | DeleteNodeCommand |
| GetNodeUseCase | 获取节点详情 | GetNodeQuery |
| ListNodesUseCase | 列出节点 | ListNodesQuery |
| GenerateNodeTokenUseCase | 生成节点 Token | GenerateNodeTokenCommand |
| ActivateNodeUseCase | 激活节点 | ActivateNodeCommand |
| DeactivateNodeUseCase | 停用节点 | DeactivateNodeCommand |

### 节点组管理用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| CreateNodeGroupUseCase | 创建节点组 | CreateNodeGroupCommand |
| UpdateNodeGroupUseCase | 更新节点组 | UpdateNodeGroupCommand |
| DeleteNodeGroupUseCase | 删除节点组 | DeleteNodeGroupCommand |
| AddNodeToGroupUseCase | 添加节点到组 | AddNodeToGroupCommand |
| RemoveNodeFromGroupUseCase | 从组移除节点 | RemoveNodeFromGroupCommand |
| AssociateGroupWithPlanUseCase | 关联订阅计划 | AssociateGroupWithPlanCommand |
| ListNodeGroupsUseCase | 列出节点组 | ListNodeGroupsQuery |

### 订阅生成用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| GenerateSubscriptionUseCase | 生成订阅 | GenerateSubscriptionCommand |
| ValidateSubscriptionAccessUseCase | 验证订阅访问 | ValidateSubscriptionAccessCommand |

### 节点上报用例

| 用例 | 描述 | 命令 |
|------|------|------|
| ReportNodeDataUseCase | 处理节点数据上报 | ReportNodeDataCommand |
| ValidateNodeTokenUseCase | 验证节点 Token | ValidateNodeTokenCommand |

### 流量管理用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| RecordNodeTrafficUseCase | 记录节点流量 | RecordNodeTrafficCommand |
| GetNodeTrafficStatsUseCase | 获取流量统计 | GetNodeTrafficStatsQuery |
| CheckTrafficLimitUseCase | 检查流量限制 | CheckTrafficLimitQuery |
| ResetNodeTrafficUseCase | 重置节点流量 | ResetNodeTrafficCommand |

## 使用示例

### 1. 创建节点并生成 Token

```go
// 管理员创建节点
createNodeUC := usecases.NewCreateNodeUseCase(
    nodeRepo,
    logger,
)

cmd := usecases.CreateNodeCommand{
    Name:          "US-01",
    ServerAddress: "node1.example.com",
    ServerPort:    8388,
    Method:        "aes-256-gcm",
    Password:      "secure_password_123",
    Plugin: &PluginConfigDTO{
        Plugin: "obfs-local",
        Opts: map[string]string{
            "obfs":      "http",
            "obfs-host": "www.bing.com",
        },
    },
    Country:      "US",
    Region:       "California",
    Tags:         []string{"premium", "gaming"},
    Description:  "美国洛杉矶高速节点",
    MaxUsers:     0, // 无限制
    TrafficLimit: 0, // 无限制
    SortOrder:    1,
}

result, err := createNodeUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}

// 保存 Token（仅此一次显示）
fmt.Println("节点创建成功！")
fmt.Println("节点 ID:", result.Node.ID)
fmt.Println("请妥善保存节点 Token:")
fmt.Println(result.APIToken)
// node_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 2. 创建节点组并关联订阅计划

```go
// 创建节点组
createGroupUC := usecases.NewCreateNodeGroupUseCase(
    nodeGroupRepo,
    logger,
)

groupCmd := usecases.CreateNodeGroupCommand{
    Name:        "美国节点组",
    Description: "包含所有美国地区的节点",
    IsPublic:    true,
    SortOrder:   1,
}

group, err := createGroupUC.Execute(ctx, groupCmd)
if err != nil {
    log.Fatal(err)
}

// 添加节点到组
addNodeUC := usecases.NewAddNodeToGroupUseCase(
    nodeGroupRepo,
    nodeRepo,
    logger,
)

addCmd := usecases.AddNodeToGroupCommand{
    GroupID: group.ID,
    NodeID:  result.Node.ID,
}

err = addNodeUC.Execute(ctx, addCmd)
if err != nil {
    log.Fatal(err)
}

// 关联订阅计划
associateUC := usecases.NewAssociateGroupWithPlanUseCase(
    nodeGroupRepo,
    planRepo,
    logger,
)

associateCmd := usecases.AssociateGroupWithPlanCommand{
    GroupID: group.ID,
    PlanID:  proPlanID, // 专业版计划 ID
}

err = associateUC.Execute(ctx, associateCmd)
```

### 3. 节点程序上报数据

参见前面的 Python 示例代码。

### 4. 生成订阅链接

```go
// 用户通过订阅令牌访问订阅
generateSubUC := usecases.NewGenerateSubscriptionUseCase(
    subscriptionRepo,
    nodeGroupRepo,
    nodeRepo,
    logger,
)

// Base64 格式
cmd := usecases.GenerateSubscriptionCommand{
    SubscriptionToken: "sub_token_xxxxxxxxxx",
    Format:            "base64",
    UserAgent:         "Shadowsocks/1.0",
}

result, err := generateSubUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}

fmt.Println("订阅内容 (Base64):")
fmt.Println(result.Content)
fmt.Println("节点数量:", result.Nodes)

// Clash 格式
cmd.Format = "clash"
result, err = generateSubUC.Execute(ctx, cmd)
fmt.Println("订阅内容 (Clash YAML):")
fmt.Println(result.Content)

// SIP008 格式
cmd.Format = "sip008"
result, err = generateSubUC.Execute(ctx, cmd)
fmt.Println("订阅内容 (SIP008 JSON):")
fmt.Println(result.Content)
```

### 5. HTTP API 端点示例

```go
// 订阅 URL 端点
func SetupSubscriptionRoutes(router *gin.Engine) {
    router.GET("/sub/:token", subscriptionHandler.GetSubscription)
    router.GET("/sub/:token/clash", subscriptionHandler.GetClashSubscription)
    router.GET("/sub/:token/v2ray", subscriptionHandler.GetV2RaySubscription)
    router.GET("/sub/:token/sip008", subscriptionHandler.GetSIP008Subscription)
    router.GET("/sub/:token/surge", subscriptionHandler.GetSurgeSubscription)
}

// 处理订阅请求
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
    token := c.Param("token")
    format := c.DefaultQuery("format", "base64")

    cmd := usecases.GenerateSubscriptionCommand{
        SubscriptionToken: token,
        Format:            format,
        UserAgent:         c.GetHeader("User-Agent"),
    }

    result, err := h.generateSubUC.Execute(c.Request.Context(), cmd)
    if err != nil {
        c.JSON(403, gin.H{"error": "invalid subscription"})
        return
    }

    // 设置响应头
    c.Header("Content-Type", "text/plain; charset=utf-8")
    c.Header("Subscription-Userinfo", fmt.Sprintf("upload=0; download=0; total=%d", result.Nodes))
    c.Header("Profile-Update-Interval", "3600") // 1 小时更新一次

    c.String(200, result.Content)
}
```

### 6. 查询流量统计

```go
// 查询节点流量统计
getStatsUC := usecases.NewGetNodeTrafficStatsUseCase(
    trafficRepo,
    logger,
)

query := usecases.TrafficStatsQuery{
    NodeID:      &nodeID,
    StartTime:   time.Now().AddDate(0, 0, -7), // 最近 7 天
    EndTime:     time.Now(),
    Granularity: "day", // 按天聚合
}

stats, err := getStatsUC.Execute(ctx, query)
if err != nil {
    log.Fatal(err)
}

fmt.Println("节点流量统计 (最近 7 天):")
for _, stat := range stats {
    fmt.Printf("日期: %s, 上传: %d MB, 下载: %d MB, 总计: %d MB\n",
        stat.Period.Format("2006-01-02"),
        stat.Upload/1024/1024,
        stat.Download/1024/1024,
        stat.Total/1024/1024)
}

// 查询用户流量统计
userQuery := usecases.TrafficStatsQuery{
    UserID:      &userID,
    StartTime:   time.Now().AddDate(0, -1, 0), // 最近一个月
    EndTime:     time.Now(),
    Granularity: "day",
}

userStats, err := getStatsUC.Execute(ctx, userQuery)
// ...
```

## 与订阅系统集成

### 1. 订阅计划关联节点组

在订阅计划中配置可访问的节点组：

```go
// 在 Subscription Domain
type SubscriptionPlan struct {
    // ... 其他字段
    AllowedNodeGroups []uint // 可访问的节点组 ID 列表
}

// 创建订阅计划时关联节点组
createPlanCmd := subscription.CreateSubscriptionPlanCommand{
    Name:              "专业版",
    Slug:              "pro",
    Price:             9900,
    AllowedNodeGroups: []uint{1, 2, 3}, // 允许访问节点组 1、2、3
    // ...
}
```

### 2. 订阅令牌验证

生成订阅时验证用户权限：

```go
func (uc *GenerateSubscriptionUseCase) Execute(ctx context.Context, cmd GenerateSubscriptionCommand) (*GenerateSubscriptionResult, error) {
    // 1. 验证订阅令牌
    subscription, err := uc.subscriptionService.ValidateToken(ctx, cmd.SubscriptionToken)
    if err != nil {
        return nil, fmt.Errorf("invalid subscription token: %w", err)
    }

    // 2. 检查订阅状态
    if !subscription.IsActive() {
        return nil, fmt.Errorf("subscription is not active")
    }

    // 3. 获取订阅计划
    plan, err := uc.planRepo.GetByID(ctx, subscription.PlanID)
    if err != nil {
        return nil, err
    }

    // 4. 获取允许访问的节点组
    var allNodes []*Node
    for _, groupID := range plan.AllowedNodeGroups {
        nodes, err := uc.nodeRepo.GetByGroupID(ctx, groupID)
        if err != nil {
            continue
        }
        allNodes = append(allNodes, nodes...)
    }

    // 5. 过滤激活状态的节点
    activeNodes := filterActiveNodes(allNodes)

    // 6. 生成订阅内容
    content := uc.generateContent(activeNodes, cmd.Format)

    // 7. 记录订阅生成事件
    uc.eventPublisher.Publish(SubscriptionGeneratedEvent{
        SubscriptionID: subscription.ID,
        UserID:         subscription.UserID,
        Format:         cmd.Format,
        NodeCount:      len(activeNodes),
        Timestamp:      time.Now(),
    })

    return &GenerateSubscriptionResult{
        Format:  cmd.Format,
        Content: content,
        Nodes:   len(activeNodes),
    }, nil
}
```

### 3. 流量限制集成

基于订阅计划的流量限制：

```go
// 订阅计划配置流量限制
type SubscriptionPlan struct {
    // ...
    MonthlyTrafficLimit uint64 // 每月流量限制（字节）
}

// 检查用户流量是否超限
func (uc *CheckTrafficLimitUseCase) Execute(ctx context.Context, userID uint) (bool, error) {
    // 1. 获取用户订阅
    subscription, err := uc.subscriptionRepo.GetActiveByUserID(ctx, userID)
    if err != nil {
        return false, err
    }

    // 2. 获取订阅计划
    plan, err := uc.planRepo.GetByID(ctx, subscription.PlanID)
    if err != nil {
        return false, err
    }

    // 3. 如果没有流量限制，返回 false
    if plan.MonthlyTrafficLimit == 0 {
        return false, nil
    }

    // 4. 获取本月流量使用情况
    startOfMonth := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -time.Now().Day()+1)
    endOfMonth := startOfMonth.AddDate(0, 1, 0)

    query := TrafficStatsQuery{
        UserID:    &userID,
        StartTime: startOfMonth,
        EndTime:   endOfMonth,
    }

    stats, err := uc.trafficRepo.GetTrafficStats(ctx, query)
    if err != nil {
        return false, err
    }

    // 5. 计算总流量
    var totalTraffic uint64
    for _, stat := range stats {
        totalTraffic += stat.Total
    }

    // 6. 检查是否超限
    return totalTraffic >= plan.MonthlyTrafficLimit, nil
}

// 节点访问时检查流量
func NodeAccessMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetUint("user_id")

        exceeded, err := checkTrafficLimitUC.Execute(c.Request.Context(), userID)
        if err != nil {
            c.JSON(500, gin.H{"error": "failed to check traffic limit"})
            c.Abort()
            return
        }

        if exceeded {
            c.JSON(403, gin.H{
                "error": "monthly traffic limit exceeded",
                "message": "请升级订阅计划或等待下月流量重置",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 4. 订阅过期处理

监听订阅过期事件，撤销节点访问权限：

```go
// 订阅过期事件处理器
type SubscriptionExpiredEventHandler struct {
    subscriptionRepo SubscriptionRepository
    logger           Logger
}

func (h *SubscriptionExpiredEventHandler) Handle(event SubscriptionExpiredEvent) {
    ctx := context.Background()

    h.logger.Info("subscription expired",
        "subscription_id", event.SubscriptionID,
        "user_id", event.UserID)

    // 订阅过期后，用户访问订阅 URL 时会自动验证失败
    // 无需额外处理，因为验证逻辑已经检查订阅状态

    // 可选：发送过期通知邮件
    // h.emailService.SendSubscriptionExpiredEmail(event.UserID)
}
```

### 5. 节点访问日志记录

记录用户访问节点的详细日志：

```go
// 节点连接时记录访问日志
func RecordNodeAccess(ctx context.Context, nodeID, userID, subscriptionID uint, clientIP string) error {
    log := &NodeAccessLog{
        NodeID:         nodeID,
        UserID:         userID,
        SubscriptionID: subscriptionID,
        ClientIP:       clientIP,
        ConnectTime:    time.Now(),
    }

    return accessLogRepo.Create(log)
}

// 统计用户访问节点频率
func GetUserAccessStats(ctx context.Context, userID uint, days int) (*AccessStats, error) {
    startTime := time.Now().AddDate(0, 0, -days)

    filter := AccessLogFilter{
        UserID:    &userID,
        StartTime: &startTime,
    }

    logs, _, err := accessLogRepo.List(ctx, filter)
    if err != nil {
        return nil, err
    }

    // 统计各节点访问次数
    nodeAccessCount := make(map[uint]int)
    for _, log := range logs {
        nodeAccessCount[log.NodeID]++
    }

    return &AccessStats{
        TotalAccess:     len(logs),
        NodeAccessCount: nodeAccessCount,
    }, nil
}
```

## 安全特性

### 1. 订阅链接安全

#### URL 签名

为防止订阅地址泄露和滥用，对订阅 URL 进行签名：

```go
type SubscriptionURLSigner struct {
    secret []byte
}

func NewSubscriptionURLSigner(secret string) *SubscriptionURLSigner {
    return &SubscriptionURLSigner{
        secret: []byte(secret),
    }
}

// 生成签名的订阅 URL
func (s *SubscriptionURLSigner) GenerateURL(subscriptionToken string, expiresAt time.Time) string {
    // 1. 构造签名数据
    data := fmt.Sprintf("%s:%d", subscriptionToken, expiresAt.Unix())

    // 2. 计算 HMAC-SHA256
    mac := hmac.New(sha256.New, s.secret)
    mac.Write([]byte(data))
    signature := hex.EncodeToString(mac.Sum(nil))

    // 3. 构造 URL
    url := fmt.Sprintf("https://api.example.com/sub/%s?expires=%d&sig=%s",
        subscriptionToken,
        expiresAt.Unix(),
        signature)

    return url
}

// 验证签名
func (s *SubscriptionURLSigner) Verify(subscriptionToken string, expiresAt int64, signature string) bool {
    // 1. 检查是否过期
    if time.Now().Unix() > expiresAt {
        return false
    }

    // 2. 重新计算签名
    data := fmt.Sprintf("%s:%d", subscriptionToken, expiresAt)
    mac := hmac.New(sha256.New, s.secret)
    mac.Write([]byte(data))
    expectedSig := hex.EncodeToString(mac.Sum(nil))

    // 3. 恒定时间比较
    return subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) == 1
}
```

#### 防重放攻击

```go
// 订阅请求中间件
func SubscriptionSecurityMiddleware(signer *SubscriptionURLSigner) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.Param("token")
        expiresStr := c.Query("expires")
        signature := c.Query("sig")

        // 验证参数
        if expiresStr == "" || signature == "" {
            c.JSON(400, gin.H{"error": "missing security parameters"})
            c.Abort()
            return
        }

        expires, err := strconv.ParseInt(expiresStr, 10, 64)
        if err != nil {
            c.JSON(400, gin.H{"error": "invalid expires parameter"})
            c.Abort()
            return
        }

        // 验证签名
        if !signer.Verify(token, expires, signature) {
            c.JSON(403, gin.H{"error": "invalid signature or expired"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 2. 节点 Token 认证

节点 Token 使用 SHA256 哈希存储，防止泄露：

```go
// 生成节点 Token
func GenerateNodeToken() (plainToken, tokenHash string, err error) {
    // 生成 32 字节随机 Token
    tokenBytes := make([]byte, 32)
    _, err = rand.Read(tokenBytes)
    if err != nil {
        return "", "", err
    }

    plainToken = "node_" + base64.URLEncoding.EncodeToString(tokenBytes)

    // 计算 SHA256 哈希
    hash := sha256.Sum256([]byte(plainToken))
    tokenHash = hex.EncodeToString(hash[:])

    return plainToken, tokenHash, nil
}

// 验证节点 Token
func VerifyNodeToken(plainToken, storedHash string) bool {
    // 计算输入 Token 的哈希
    hash := sha256.Sum256([]byte(plainToken))
    tokenHash := hex.EncodeToString(hash[:])

    // 恒定时间比较，防止时序攻击
    return subtle.ConstantTimeCompare([]byte(tokenHash), []byte(storedHash)) == 1
}
```

### 3. 流量审计

详细记录节点访问和流量使用情况：

```go
// 审计日志结构
type AuditLog struct {
    ID             uint
    EventType      string // subscription_access, node_report, traffic_exceeded
    UserID         *uint
    NodeID         *uint
    SubscriptionID *uint
    Details        string // JSON
    IPAddress      string
    UserAgent      string
    CreatedAt      time.Time
}

// 记录订阅访问
func AuditSubscriptionAccess(ctx context.Context, subscription *Subscription, clientIP, userAgent string) {
    log := &AuditLog{
        EventType:      "subscription_access",
        UserID:         &subscription.UserID,
        SubscriptionID: &subscription.ID,
        Details: toJSON(map[string]interface{}{
            "format": "base64",
            "nodes":  10,
        }),
        IPAddress: clientIP,
        UserAgent: userAgent,
        CreatedAt: time.Now(),
    }

    auditRepo.Create(log)
}

// 异常流量告警
func CheckAbnormalTraffic(ctx context.Context, nodeID uint, traffic uint64) {
    // 获取节点历史平均流量
    avgTraffic := getNodeAverageTraffic(ctx, nodeID)

    // 如果流量超过平均值 5 倍，触发告警
    if traffic > avgTraffic*5 {
        log := &AuditLog{
            EventType: "abnormal_traffic",
            NodeID:    &nodeID,
            Details: toJSON(map[string]interface{}{
                "traffic":     traffic,
                "avg_traffic": avgTraffic,
                "ratio":       float64(traffic) / float64(avgTraffic),
            }),
            CreatedAt: time.Now(),
        }

        auditRepo.Create(log)

        // 发送告警通知
        alertService.SendAlert("异常流量检测", fmt.Sprintf("节点 %d 流量异常", nodeID))
    }
}
```

### 4. 订阅 URL 限流

防止订阅地址泄露导致的滥用：

```go
// 基于 Redis 的限流器
type SubscriptionRateLimiter struct {
    client *redis.Client
}

func (l *SubscriptionRateLimiter) Allow(subscriptionID uint, ip string) (bool, error) {
    // 限制：每个订阅每分钟最多 10 次访问
    key := fmt.Sprintf("ratelimit:subscription:%d:%s", subscriptionID, ip)

    count, err := l.client.Incr(context.Background(), key).Result()
    if err != nil {
        return false, err
    }

    if count == 1 {
        // 第一次访问，设置过期时间
        l.client.Expire(context.Background(), key, time.Minute)
    }

    return count <= 10, nil
}

// 限流中间件
func SubscriptionRateLimitMiddleware(limiter *SubscriptionRateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        subscriptionID := c.GetUint("subscription_id")
        clientIP := c.ClientIP()

        allowed, err := limiter.Allow(subscriptionID, clientIP)
        if err != nil {
            c.JSON(500, gin.H{"error": "rate limit check failed"})
            c.Abort()
            return
        }

        if !allowed {
            c.JSON(429, gin.H{
                "error":   "rate limit exceeded",
                "message": "请求过于频繁，请稍后再试",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

## 最佳实践

### 1. 节点命名规范

采用统一的命名格式，便于管理和识别：

```
格式: [国家代码]-[地区简称]-[序号]

示例:
- US-LA-01  (美国洛杉矶 01 号)
- JP-TK-01  (日本东京 01 号)
- HK-HK-01  (香港 01 号)
- SG-SG-01  (新加坡 01 号)

特殊标记:
- US-LA-01-Premium  (高级节点)
- JP-TK-01-Gaming   (游戏专用)
- HK-HK-01-Lite     (轻量节点)
```

### 2. 节点分组策略

合理的分组策略可以提高管理效率：

**按地区分组：**
- 美国节点组
- 亚洲节点组
- 欧洲节点组

**按性能分组：**
- 高速节点组（Premium）
- 标准节点组（Standard）
- 轻量节点组（Lite）

**按用途分组：**
- 游戏专用节点组
- 流媒体节点组
- 通用节点组

**按订阅计划分组：**
- 基础版节点组
- 专业版节点组
- 企业版节点组

### 3. 订阅更新频率

**建议客户端更新频率：**
- 自动更新间隔：1-4 小时
- 手动更新：用户主动触发
- 节点变更通知：服务端推送

**实现方式：**
```yaml
# Clash 配置
profile:
  store-selected: true
  store-fake-ip: true
  update-interval: 3600  # 1 小时自动更新
```

**服务端优化：**
- 缓存订阅内容（5-10 分钟）
- 使用 CDN 加速订阅下载
- 提供配置版本号，客户端对比是否需要更新

### 4. 流量统计精度

**上报频率：**
- 节点上报：30 秒批量上报
- 实时累计：更新节点流量使用
- 按小时聚合：详细统计数据
- 按天汇总：历史趋势分析
- 按月汇总：账单和报表

**存储策略：**
```
原始数据（按小时）: 保留 30 天
按天聚合数据:      保留 1 年
按月聚合数据:      永久保留
```

### 5. 性能优化建议

#### 订阅生成优化

```go
// 缓存订阅内容
type SubscriptionCache struct {
    cache *redis.Client
    ttl   time.Duration
}

func (sc *SubscriptionCache) Get(subscriptionID uint, format string) (string, error) {
    key := fmt.Sprintf("subscription:%d:%s", subscriptionID, format)
    return sc.cache.Get(context.Background(), key).Result()
}

func (sc *SubscriptionCache) Set(subscriptionID uint, format, content string) error {
    key := fmt.Sprintf("subscription:%d:%s", subscriptionID, format)
    return sc.cache.Set(context.Background(), key, content, sc.ttl).Err()
}

// 使用缓存的订阅生成
func (uc *GenerateSubscriptionUseCase) ExecuteWithCache(ctx context.Context, cmd GenerateSubscriptionCommand) (*GenerateSubscriptionResult, error) {
    // 尝试从缓存获取
    cached, err := uc.cache.Get(cmd.SubscriptionID, cmd.Format)
    if err == nil {
        return &GenerateSubscriptionResult{
            Format:  cmd.Format,
            Content: cached,
            Cached:  true,
        }, nil
    }

    // 缓存未命中，生成订阅
    result, err := uc.Execute(ctx, cmd)
    if err != nil {
        return nil, err
    }

    // 缓存结果
    uc.cache.Set(cmd.SubscriptionID, cmd.Format, result.Content)

    return result, nil
}
```

#### 流量数据批量写入

```go
// 批量写入流量数据
type TrafficBatchWriter struct {
    buffer   []*NodeTraffic
    mu       sync.Mutex
    repo     NodeTrafficRepository
    interval time.Duration
}

func (bw *TrafficBatchWriter) Add(traffic *NodeTraffic) {
    bw.mu.Lock()
    defer bw.mu.Unlock()

    bw.buffer = append(bw.buffer, traffic)
}

func (bw *TrafficBatchWriter) Flush() error {
    bw.mu.Lock()
    defer bw.mu.Unlock()

    if len(bw.buffer) == 0 {
        return nil
    }

    // 批量写入
    err := bw.repo.BatchCreate(context.Background(), bw.buffer)
    if err != nil {
        return err
    }

    // 清空缓冲区
    bw.buffer = bw.buffer[:0]
    return nil
}

func (bw *TrafficBatchWriter) Run() {
    ticker := time.NewTicker(bw.interval)
    for range ticker.C {
        bw.Flush()
    }
}
```

### 6. 节点 Token 轮换策略

定期轮换节点 Token，提高安全性：

```go
// Token 轮换策略
type TokenRotationPolicy struct {
    interval time.Duration // 轮换间隔，如 90 天
}

// 检查是否需要轮换
func (p *TokenRotationPolicy) ShouldRotate(node *Node) bool {
    // 检查 Token 生成时间
    tokenAge := time.Since(node.TokenGeneratedAt)
    return tokenAge >= p.interval
}

// 定时任务：轮换过期 Token
func RotateExpiredTokens() {
    ctx := context.Background()
    policy := TokenRotationPolicy{interval: 90 * 24 * time.Hour}

    nodes, _ := nodeRepo.List(ctx, NodeFilter{})

    for _, node := range nodes {
        if policy.ShouldRotate(node) {
            // 生成新 Token
            plainToken, tokenHash, _ := GenerateNodeToken()

            node.TokenHash = tokenHash
            node.TokenGeneratedAt = time.Now()

            nodeRepo.Update(ctx, node)

            // 通知管理员新 Token
            notificationService.NotifyAdmins(
                "节点 Token 已轮换",
                fmt.Sprintf("节点 %s 的 Token 已自动轮换，新 Token: %s", node.Name, plainToken),
            )
        }
    }
}
```

## 扩展点

### 1. 多协议支持

当前仅支持 Shadowsocks，未来可扩展支持：

- **VMess** (V2Ray)
- **Trojan**
- **Hysteria** (基于 QUIC)
- **WireGuard**

**扩展设计：**

```go
// 协议抽象
type Protocol interface {
    Name() string
    GenerateURI(node *Node) string
    ToClashConfig(node *Node) map[string]interface{}
    ToV2RayConfig(node *Node) map[string]interface{}
}

// Shadowsocks 协议实现
type ShadowsocksProtocol struct{}

func (p *ShadowsocksProtocol) Name() string {
    return "shadowsocks"
}

func (p *ShadowsocksProtocol) GenerateURI(node *Node) string {
    // ss://...
}

// VMess 协议实现
type VMess Protocol struct{}

func (p *VMessProtocol) Name() string {
    return "vmess"
}

func (p *VMessProtocol) GenerateURI(node *Node) string {
    // vmess://...
}

// 节点支持多协议
type Node struct {
    // ...
    Protocol Protocol
}
```

### 2. 节点健康检查

定期检查节点连通性和延迟：

```go
type NodeHealthChecker struct {
    nodeRepo NodeRepository
    interval time.Duration
}

func (hc *NodeHealthChecker) Check(node *Node) *HealthCheckResult {
    // 1. TCP 连通性检查
    conn, err := net.DialTimeout("tcp",
        fmt.Sprintf("%s:%d", node.ServerAddress.Value(), node.ServerPort),
        5*time.Second)

    if err != nil {
        return &HealthCheckResult{
            NodeID:      node.ID,
            IsHealthy:   false,
            Error:       err.Error(),
            CheckedAt:   time.Now(),
        }
    }
    conn.Close()

    // 2. 延迟测试
    start := time.Now()
    // 发送测试数据...
    latency := time.Since(start)

    return &HealthCheckResult{
        NodeID:    node.ID,
        IsHealthy: true,
        Latency:   latency,
        CheckedAt: time.Now(),
    }
}

func (hc *NodeHealthChecker) Run() {
    ticker := time.NewTicker(hc.interval)
    for range ticker.C {
        nodes, _ := hc.nodeRepo.GetByStatus(context.Background(), NodeStatusActive)

        for _, node := range nodes {
            result := hc.Check(node)

            if !result.IsHealthy {
                // 节点不健康，触发告警
                // 可选：自动停用节点
                node.EnterMaintenance("health check failed")
                hc.nodeRepo.Update(context.Background(), node)
            }
        }
    }
}
```

### 3. 智能路由规则

根据用户地理位置推荐最优节点：

```go
type SmartRouter struct {
    geoipDB *geoip.DB
}

func (sr *SmartRouter) RecommendNodes(clientIP string, nodes []*Node) []*Node {
    // 1. 获取用户地理位置
    userCountry := sr.geoipDB.Lookup(clientIP).Country

    // 2. 按距离排序节点
    scored := make([]struct {
        node  *Node
        score int
    }, len(nodes))

    for i, node := range nodes {
        score := 0

        // 同国家节点优先
        if node.Metadata.Country() == userCountry {
            score += 100
        }

        // 延迟越低分数越高
        if node.Latency > 0 {
            score += int(1000 / node.Latency.Milliseconds())
        }

        // 负载越低分数越高
        if node.OnlineUsers < node.MaxUsers {
            score += int(100 * (1 - float64(node.OnlineUsers)/float64(node.MaxUsers)))
        }

        scored[i] = struct {
            node  *Node
            score int
        }{node, score}
    }

    // 3. 排序
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    // 4. 返回推荐节点
    recommended := make([]*Node, len(scored))
    for i, s := range scored {
        recommended[i] = s.node
    }

    return recommended
}
```

### 4. 节点负载均衡

自动选择负载较低的节点：

```go
type LoadBalancer struct {
    strategy LoadBalanceStrategy
}

type LoadBalanceStrategy interface {
    Select(nodes []*Node) *Node
}

// 轮询策略
type RoundRobinStrategy struct {
    current int
    mu      sync.Mutex
}

func (rr *RoundRobinStrategy) Select(nodes []*Node) *Node {
    rr.mu.Lock()
    defer rr.mu.Unlock()

    if len(nodes) == 0 {
        return nil
    }

    node := nodes[rr.current]
    rr.current = (rr.current + 1) % len(nodes)
    return node
}

// 最少连接策略
type LeastConnectionStrategy struct{}

func (lc *LeastConnectionStrategy) Select(nodes []*Node) *Node {
    if len(nodes) == 0 {
        return nil
    }

    minNode := nodes[0]
    for _, node := range nodes[1:] {
        if node.OnlineUsers < minNode.OnlineUsers {
            minNode = node
        }
    }

    return minNode
}

// 加权轮询策略
type WeightedRoundRobinStrategy struct {
    weights map[uint]int
}

func (wrr *WeightedRoundRobinStrategy) Select(nodes []*Node) *Node {
    // 根据权重选择节点
    // ...
}
```

## RBAC 权限定义

### 节点资源 (node)

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| node:create | 创建节点 | admin |
| node:read | 查看节点 | admin |
| node:update | 更新节点 | admin |
| node:delete | 删除节点 | admin |
| node:generate_token | 生成节点 Token | admin |
| node:view_token | 查看节点 Token | admin |
| node:activate | 激活节点 | admin |
| node:deactivate | 停用节点 | admin |

### 节点组资源 (node_group)

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| node_group:create | 创建节点组 | admin |
| node_group:read | 查看节点组 | admin |
| node_group:update | 更新节点组 | admin |
| node_group:delete | 删除节点组 | admin |
| node_group:manage_nodes | 管理组内节点 | admin |
| node_group:associate_plan | 关联订阅计划 | admin |

### 订阅资源 (subscription_url)

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| subscription_url:access | 访问订阅链接 | user, admin |
| subscription_url:generate | 生成订阅 | user, admin |

### 流量资源 (traffic)

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| traffic:view_own | 查看自己的流量统计 | user, admin |
| traffic:view_all | 查看所有流量统计 | admin |
| traffic:reset | 重置流量 | admin |

## 相关文档

- [订阅管理文档](SUBSCRIPTION_DOMAIN.md)
- [用户领域文档](USER_DOMAIN.md)
- [权限系统文档](PERMISSION_SYSTEM.md)
- [管理员分配指南](ASSIGN_ADMIN.md)

## 总结

节点管理领域提供了完整的代理节点管理和订阅服务解决方案，核心功能包括：

### 功能清单

- ✅ 完整的节点生命周期管理
- ✅ 灵活的节点分组策略
- ✅ 多格式订阅生成 (Base64/Clash/V2Ray/SIP008/Surge)
- ✅ HTTP API 节点数据上报
- ✅ Token 认证，简单可靠
- ✅ 与订阅系统深度集成
- ✅ 流量统计和限制
- ✅ 访问日志审计
- ✅ 订阅链接签名防篡改
- ✅ 基于订阅计划的节点访问控制

### 架构优势

- **DDD 设计**：领域边界清晰，聚合根封装业务逻辑
- **HTTP API**：节点上报简单可靠，易于扩展
- **Token 认证**：节点认证简单有效，哈希存储安全
- **异步处理**：流量数据异步处理，不阻塞节点
- **节点池共享**：灵活的节点分组和订阅计划关联
- **多格式支持**：兼容主流客户端，用户体验友好
- **安全可靠**：签名、限流、审计多重保障

### 技术特点

- **Go 语言实现**：高性能、并发安全
- **PostgreSQL**：可靠的数据持久化
- **Redis**：缓存和限流
- **消息队列**：异步处理流量数据
- **RESTful API**：标准化接口设计
- **YAML/JSON**：多格式订阅支持

该领域为 SaaS 代理服务提供了生产级的节点管理能力，支持大规模节点部署和用户订阅服务。
