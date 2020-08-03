# Influx Tool 说明文档

Influx Tool 是一个基于 InfluxDB 的配套工具，目前仅支持 **数据导出** 功能

### 功能

- 数据导出
  - 相当于原生 `influx_inspect export` 工具的扩展版，其导出数据兼容 `influx`，可以使用 `influx -import -path` 进行导入

### 用法

```
$ ./influx-tool -h
Usage of ./influx-tool:
  -boolean-fields string
    	fields required to cast to boolean from string, split by ','
  -database string
    	database to connect to the server
  -dir string
    	directory to export (default "export")
  -end string
    	the end unix time to export (second precision), optional
  -float-fields string
    	fields required to cast to float from string, split by ','
  -format string
    	the output format to export, valid values are line or csv (default "line")
  -host string
    	host to connect to (default "127.0.0.1")
  -integer-fields string
    	fields required to cast to integer from string, split by ','
  -measurements string
    	measurements split by ',' while return all measurements if empty
    	wildcard '*' and '?' supported
  -merge
    	merge and export into one file, ignored when -format is not line
  -password string
    	password to connect to the server
  -port int
    	port to connect to (default 8086)
  -range string
    	measurements range to export, as 'start,end', started from 1, included end
    	ignored when -measurements not empty
  -ssl
    	use https for requests
  -start string
    	the start unix time to export (second precision), optional
  -username string
    	username to connect to the server
  -version
    	display the version and exit
  -worker int
    	number of concurrent workers to export (default 1)
```

版本信息显示：

```
$ ./influx-tool -version
Version:    0.1.8
Git commit: 76e4a95
Go version: go1.14.6
Build time: 2020-08-02 17:27:33
OS/Arch:    linux/amd64
```

### 选项说明

- `-host`: 指定 influxdb 的 host，默认为 `127.0.0.1`
- `-port`: 指定 influxdb 的 port，默认为 `8086`
- `-username`: 认证的用户名，默认为空
- `-password`: 认证的密码，默认为空
- `-ssl`: 是否启用 https，默认为 `false`
- `-database`: 指定导出的数据库名称，必填
- `-measurements`: 需要导出的 measurement 列表，以英文逗号分隔，空表示导出全部列表，支持通配符 `*` 和 `?`
- `-range`: 需要导出的 measurement 列表的起止闭区间，从 1 开始计数，当 `-measurements` 非空时此选项被忽略
- `-start`: 导出数据的开始时间戳，精度为秒，未指定则没有开始时间限制
- `-end`: 导出数据的结束时间戳，精度为秒，未指定则没有结束时间限制
- `-format`: 导出数据的格式，可选值为 `line` 或 `csv`，默认 `line`，即官方默认导出的 line protocol 格式
- `-dir`: 导出的目录，默认为 `export`
- `-merge`: 当 `-format` 不为 `line` 时此选项被忽略，开启后将合并成一个文件，默认为 `false`
- `-worker`: 用于导出文件的并行工作线程数量，默认为 `1`
- `-boolean-fields`: 需要将 string 类型转为 boolean 类型的 field 列表，以英文逗号分隔
- `-float-fields`: 需要将 string 类型转为 float 类型的 field 列表，以英文逗号分隔
- `-integer-fields`: 需要将 string 类型转为 integer 类型的 field 列表，以英文逗号分隔
- `-version`: 显示版本信息

### 注意事项

#### 数据类型

- 查询语句：`SHOW FIELD KEYS`
- 同一 field 存在多种不同类型（即至少两种类型）的数据时，将默认依次按照以下规则转换
  - 如果存在 float 类型，则 boolean 类型数据丢弃，integer、string 类型转换为 float 类型
  - 如果存在 integer 类型，则 boolean 类型数据丢弃，string 类型转换为 integer 类型
  - 如果存在 boolean 类型，则 string 类型转换为 boolean 类型
- 某一 field 只有 string 一种类型，但实际类型是 float、integer 或 boolean，需要导出时转换
  - 选项：`-float-fields`、`-integer-fields` 或 `-boolean-fields`
  - 导出数据时通过上述选项指定 field 列表进行强制转换

### 更新日志

#### v0.1.0

> - 支持导出基本功能，一个 measurement 将导出一个文件，以官方默认的 line protocol 格式保存
> - 支持 -host、-port、-username、-password、-ssl、-database、-measurements、-cpu、-version 选项

#### v0.1.1

> - 支持 -dir 选项

#### v0.1.2

> - 优化导出性能

#### v0.1.3

> - 支持 -range 选项，指定导出 measurement 列表的起止闭区间

#### v0.1.4

> - 修复同一 field 同时包含 string 和其它类型时，导出数据时异常退出的问题

#### v0.1.5

> - 修复 influx -import 导入失败的问题
> - 支持 -float-fields、-integer-fields、-boolean-fields 选项将 string 类型的 field 列表强制转换为指定类型

#### v0.1.6

> - 优化同一 field 包含多种不同类型时，导出数据的转换问题，请参考[注意事项](#注意事项)中的[数据类型](#数据类型)
> - 支持 -merge 选项，将导出的多个 measurement 文件合并成一个文件

#### v0.1.7

> - 支持 -measurements 包含通配符 * 和 ?
> - 支持 -start 和 -end 选项，指定导出数据的起止时间戳
> - 支持 -format 选项，指定导出格式，支持 line protocol 和 csv 格式

#### v0.1.8

> - 提升导出数据的性能
> - -cpu 选项调整为 -worker 选项，默认值为 1
