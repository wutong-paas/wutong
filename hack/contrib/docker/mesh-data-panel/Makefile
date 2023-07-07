VERSION=5.1
image:
	docker build -t wutongpaas/plugin_mesh:${VERSION} .
test:
	docker run -e XDS_HOST_IP=192.168.1.112 -e WT_TENANT_ID=5dbea040f5cb437c82e3bf02943fb02e -e WT_PLUGIN_ID=c5618104b2aa4c508390e4f18f316500 -e WT_SERVICE_ALIAS=wt439125 --rm -it wutongpaas/plugin_mesh:1.0	