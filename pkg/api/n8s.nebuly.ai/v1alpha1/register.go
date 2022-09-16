package v1alpha1

func init() {
	SchemeBuilder.Register(&ElasticQuota{}, &ElasticQuotaList{})
	SchemeBuilder.Register(&CompositeElasticQuota{}, &CompositeElasticQuotaList{})
}
