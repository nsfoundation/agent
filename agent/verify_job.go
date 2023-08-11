package agent

import (
	"errors"
	"fmt"

	"github.com/buildkite/agent/v3/internal/pipeline"
)

var ErrNoSignature = errors.New("job had no signature to verify")

type invalidSignatureError struct {
	underlying error
}

func newInvalidSignatureError(err error) *invalidSignatureError {
	return &invalidSignatureError{underlying: err}
}

func (e *invalidSignatureError) Error() string {
	return fmt.Sprintf("invalid signature: %v", e.underlying)
}

func (e *invalidSignatureError) Unwrap() error {
	return e.underlying
}

func (r *JobRunner) verificationFailureLogs(err error, behavior string) {
	label := "WARNING"
	if behavior == VerificationBehaviourBlock {
		label = "ERROR"
	}

	r.logger.Warn("Job verification failed: %s", err.Error())
	r.logStreamer.Process([]byte(fmt.Sprintf("⚠️ %s: Job verification failed: %s\n", label, err.Error())))

	if behavior == VerificationBehaviourWarn {
		r.logger.Warn("Job will be run without verification - this is not recommended. You can change this behavior with the `job-verification-no-signature-behavior` agent configuration option.")
		r.logStreamer.Process([]byte(fmt.Sprintf("⚠️ %s: Job will be run without verification\n", label)))
	}
}

func (r *JobRunner) verifyJob(verifier pipeline.Verifier) error {
	step := r.conf.Job.Step

	if step.Matrix != nil {
		r.logger.Warn("Signing/Verification of matrix jobs is not currently supported")
		r.logger.Warn("Watch this space 👀")

		return nil
	}

	if step.Signature == nil {
		return ErrNoSignature
	}

	// Verify the signature
	if err := step.Signature.Verify(&step, verifier); err != nil {
		return newInvalidSignatureError(err)
	}

	// Now that the signature of the job's step is verified, we need to check if the fields on the job match those on the
	// step. If they don't, we need to fail the job - more or less the only reason that the job and the step would have
	// different fields would be if someone had modified the job on the backend after it was signed (aka crimes)
	signedFields := step.Signature.SignedFields
	jobFields, err := r.conf.Job.ValuesForFields(signedFields)
	if err != nil {
		return fmt.Errorf("failed to get values for fields %v on job %s: %w", signedFields, r.conf.Job.ID, err)
	}

	stepFields, err := step.ValuesForFields(signedFields)
	if err != nil {
		return fmt.Errorf("failed to get values for fields %v on step: %w", signedFields, err)
	}

	for _, field := range signedFields {
		if jobFields[field] != stepFields[field] {
			return newInvalidSignatureError(fmt.Errorf("job %q was signed with signature %q, but the value of field %q on the job (%q) does not match the value of the field on the step (%q)", r.conf.Job.ID, step.Signature.Value, field, jobFields[field], stepFields[field]))
		}
	}

	return nil
}
