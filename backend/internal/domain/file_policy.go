package domain

import "time"

type FileCategory string

const (
	FileCategoryStandard FileCategory = "standard"
	FileCategorySpecial  FileCategory = "special"
)

type FilePolicy string

const (
	FilePolicyStandardUI           FilePolicy = "standard-ui"
	FilePolicyStandardDownloadable FilePolicy = "standard-downloadable"
	FilePolicySpecial              FilePolicy = "special"
)

type FilePurpose string

const (
	FilePurposeMaterial    FilePurpose = "material"
	FilePurposePayment     FilePurpose = "payment_receipt"
	FilePurposeEvent       FilePurpose = "event"
	FilePurposePassport    FilePurpose = "passport"
	FilePurposeDiploma     FilePurpose = "diploma"
	FilePurposeProfile     FilePurpose = "profile_photo"
	FilePurposeExamWriting FilePurpose = "exam_writing"
)

const WritingRetention = 60 * 24 * time.Hour

func (c FileCategory) IsValid() bool {
	switch c {
	case FileCategoryStandard, FileCategorySpecial:
		return true
	default:
		return false
	}
}

func (c FileCategory) CanOptimize() bool {
	return c == FileCategoryStandard
}

func (c FileCategory) MustPreserveOriginalBytes() bool {
	return c == FileCategorySpecial
}

func (p FilePolicy) IsValid() bool {
	switch p {
	case FilePolicyStandardUI, FilePolicyStandardDownloadable, FilePolicySpecial:
		return true
	default:
		return false
	}
}

func (p FilePolicy) Category() FileCategory {
	if p == FilePolicySpecial {
		return FileCategorySpecial
	}
	return FileCategoryStandard
}

func InferFilePolicy(category FileCategory, purpose FilePurpose) FilePolicy {
	if category == FileCategorySpecial {
		return FilePolicySpecial
	}
	if purpose == FilePurposeProfile {
		return FilePolicyStandardUI
	}
	return FilePolicyStandardDownloadable
}

func RetentionUntil(purpose FilePurpose, uploadedAt time.Time) *time.Time {
	if purpose != FilePurposeExamWriting {
		return nil
	}

	retentionUntil := uploadedAt.Add(WritingRetention)
	return &retentionUntil
}
