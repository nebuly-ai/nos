#!/usr/bin/env bash


# Generate defaults and conversions for scheduler

DEFAULTER_GEN=${DEFAULTER_GEN:-../bin/defaulter-gen}
CONVERSION_GEN=${CONVERSION_GEN:-../bin/conversion-gen}

echo $DEFAULTER_GEN

"$DEFAULTER_GEN" --input-dirs github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/scheduler/config/v1beta3 \
  -O zz_generated.defaults \
  --go-header-file="hack/boilerplate/license.txt"
"$CONVERSION_GEN" --input-dirs github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/scheduler/config,github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/scheduler/config/v1beta3 \
  -O zz_generated.conversions \
  --go-header-file="hack/boilerplate/license.txt"
cp ${GOPATH}/src/github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/scheduler/config/v1beta3/zz_generated.defaults.go pkg/api/n8s.nebuly.ai/scheduler/config/v1beta3/zz_generated.defaults.go
cp ${GOPATH}/src/github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/scheduler/config/v1beta3/zz_generated.conversions.go pkg/api/n8s.nebuly.ai/scheduler/config/v1beta3/zz_generated.conversions.go
