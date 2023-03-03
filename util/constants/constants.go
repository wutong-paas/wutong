package constants

const (
	// Wutong -
	Wutong = "wutong"
	// DefImageRepository default private image repository
	DefImageRepository = "wutong.me"
	// WTDataLogPath -
	WTDataLogPath = "/wtdata/logs"
	// ImagePullSecretKey the key of environment IMAGE_PULL_SECRET
	ImagePullSecretKey = "IMAGE_PULL_SECRET"
	// DefOnlineImageRepository default private image repository
	DefOnlineImageRepository = "swr.cn-southwest-2.myhuaweicloud.com/wutong"
)

// Kubernetes recommended Labels
// Refer to: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/#labels
const (
	ResourceManagedByLabel     = "app.kubernetes.io/managed-by"
	ResourceInstanceLabel      = "app.kubernetes.io/instance"
	ResourceTenantIDLabel      = "tenant_id"
	ResourceTenantNameLabel    = "tenant_name"
	ResourceTenantEnvIDLabel   = "tenant_env_id"
	ResourceTenantEnvNameLabel = "tenant_env_name"
	ResourceAppIDLabel         = "app_id"
	ResourceAppNameLabel       = "app"
	ResourceVersionLabel       = "version"
	ResourceServiceIDLabel     = "service_id"
	ResourceServiceAliasLabel  = "service_alias"
)
