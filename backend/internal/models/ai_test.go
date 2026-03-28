package models

import (
	"testing"
)

func TestAIProviderTypeIsCloud(t *testing.T) {
	cloudProviders := []AIProviderType{
		AIProviderOpenRouter,
		AIProviderBedrock,
		AIProviderVertexAI,
		AIProviderOpenAI,
		AIProviderAnthropic,
	}
	for _, p := range cloudProviders {
		if !p.IsCloud() {
			t.Errorf("expected %s to be cloud", p)
		}
		if p.IsLocal() {
			t.Errorf("expected %s not to be local", p)
		}
	}
}

func TestAIProviderTypeIsLocal(t *testing.T) {
	localProviders := []AIProviderType{
		AIProviderOllama,
		AIProviderLlamaCpp,
		AIProviderLMStudio,
		AIProviderVLLM,
		AIProviderLocalAI,
	}
	for _, p := range localProviders {
		if !p.IsLocal() {
			t.Errorf("expected %s to be local", p)
		}
		if p.IsCloud() {
			t.Errorf("expected %s not to be cloud", p)
		}
	}
}

func TestAIProviderTypeValid(t *testing.T) {
	validProviders := []AIProviderType{
		AIProviderOpenRouter, AIProviderBedrock, AIProviderVertexAI,
		AIProviderOpenAI, AIProviderAnthropic, AIProviderOllama,
		AIProviderLlamaCpp, AIProviderLMStudio, AIProviderVLLM, AIProviderLocalAI,
	}
	for _, p := range validProviders {
		if !p.Valid() {
			t.Errorf("expected %s to be valid", p)
		}
	}

	invalid := AIProviderType("nonexistent")
	if invalid.Valid() {
		t.Error("expected nonexistent provider to be invalid")
	}
	if invalid.IsCloud() {
		t.Error("expected nonexistent provider not to be cloud")
	}
}

func TestAIFeatureTypeValid(t *testing.T) {
	validFeatures := []AIFeatureType{
		AIFeatureSummary, AIFeatureSearch, AIFeatureChat,
		AIFeatureEmbed, AIFeatureModerate, AIFeatureTranslate,
	}
	for _, f := range validFeatures {
		if !f.Valid() {
			t.Errorf("expected %s to be valid", f)
		}
	}

	invalid := AIFeatureType("nonexistent")
	if invalid.Valid() {
		t.Error("expected nonexistent feature to be invalid")
	}
}
