package common

var AzureDeploy string = `
param
(
    [string]$resource,
    [string]$registry,
    [string]$tag,
    [string]$rUser,
    [string]$rPassword
	[string]$dbName
	[string]$dbUser
	[string]$dbPassword
)
$protocol="http://"
$sep="/"
Write-Output 'Creating web app...'
az group create --name $resource --location southcentralus
#az webapp up -n $resourceGroup --runtime 'PYTHON:3.7'  -p AppServicePlan --sku B1  --resource-group $resourceGroup
# Create a web app.
az appservice plan create --name AppServiceLinuxDockerPlan --resource-group $resource --location southcentralus --is-linux --sku B1
az webapp create --name $resource --plan AppServiceLinuxDockerPlan --resource-group $resource --deployment-container-image-name $registry$sep$tag
# Configure web app with a custom Docker Container from Azure Container Registry.
az webapp config container set --resource-group $resource --name $resource --docker-registry-server-url $protocol$registry --docker-registry-server-user $rUser --docker-registry-server-password $rPassword
Write-Output 'Created web app...'
az postgres up --server-name $resource --resource-group $resource --sku-name GP_Gen5_2 --location southcentralus --admin-user $dbUser --admin-password $dbPassword --database-name $dbName
Write-Output 'Database created with details...'
Write-Output $dbUser
Write-Output $dbPassword
Write-Output 'Found env file in root, converting to Appsettings...'
az webapp config appsettings set -g $resource -n $resource --settings "@.\.env.properties"
`

type Array struct {
	Self []interface{}
}

func (a *Array) Except(other *Array, comparer func(a, b interface{}) bool) Array {
	except := make([]interface{}, 0)
	for item := range a.Self {
		for otherItem := range other.Self {
			if !comparer(item, otherItem) {
				except = append(except, item)
			}
		}
	}
	return Array{Self: except}
}
