# hack/ README.md

## Edit histories

1. gateway/nginxtmp/servers.tmpl 模版修改 access_log：

第 87 行， dingpeng 2022/11/24

原配置

```tmpl
        {{ if $loc.DisableAccessLog }}
        access_log off;
        {{ else if $loc.AccessLogPath }}
        access_log {{$loc.AccessLogPath}} proxy;
        {{ end }}
```
