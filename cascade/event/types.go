package event

import (
	"context"
	"time"
)

// EventType represents the type of event emitted by the SuperNode SDK.
type EventType string

// Event types mirrored from the SuperNode SDK.
const (
	SDKTaskStarted            EventType = "sdk:started"
	SDKSupernodesUnavailable  EventType = "sdk:supernodes_unavailable"
	SDKSupernodesFound        EventType = "sdk:supernodes_found"
	SDKRegistrationAttempt    EventType = "sdk:registration_attempt"
	SDKRegistrationFailure    EventType = "sdk:registration_failure"
	SDKRegistrationSuccessful EventType = "sdk:registration_successful"
	SDKTaskTxHashReceived     EventType = "sdk:txhash_received"
	SDKTaskCompleted          EventType = "sdk:completed"
	SDKTaskFailed             EventType = "sdk:failed"
	SDKConnectionEstablished  EventType = "sdk:connection_established"

	SDKUploadStarted     EventType = "sdk:upload_started"
	SDKUploadCompleted   EventType = "sdk:upload_completed"
	SDKUploadFailed      EventType = "sdk:upload_failed"
	SDKProcessingStarted EventType = "sdk:processing_started"
	SDKProcessingFailed  EventType = "sdk:processing_failed"
	SDKProcessingTimeout EventType = "sdk:processing_timeout"

	SDKDownloadAttempt   EventType = "sdk:download_attempt"
	SDKDownloadFailure   EventType = "sdk:download_failure"
	SDKDownloadStarted   EventType = "sdk:download_started"
	SDKDownloadCompleted EventType = "sdk:download_completed"

	SupernodeActionRetrieved          EventType = "supernode:action_retrieved"
	SupernodeActionFeeVerified        EventType = "supernode:action_fee_verified"
	SupernodeTopCheckPassed           EventType = "supernode:top_check_passed"
	SupernodeMetadataDecoded          EventType = "supernode:metadata_decoded"
	SupernodeDataHashVerified         EventType = "supernode:data_hash_verified"
	SupernodeInputEncoded             EventType = "supernode:input_encoded"
	SupernodeSignatureVerified        EventType = "supernode:signature_verified"
	SupernodeRQIDGenerated            EventType = "supernode:rqid_generated"
	SupernodeRQIDVerified             EventType = "supernode:rqid_verified"
	SupernodeFinalizeSimulated        EventType = "supernode:finalize_simulated"
	SupernodeArtefactsStored          EventType = "supernode:artefacts_stored"
	SupernodeActionFinalized          EventType = "supernode:action_finalized"
	SupernodeArtefactsDownloaded      EventType = "supernode:artefacts_downloaded"
	SupernodeNetworkRetrieveStarted   EventType = "supernode:network_retrieve_started"
	SupernodeDecodeCompleted          EventType = "supernode:decode_completed"
	SupernodeServeReady               EventType = "supernode:serve_ready"
	SupernodeUnknown                  EventType = "supernode:unknown"
	SupernodeFinalizeSimulationFailed EventType = "supernode:finalize_simulation_failed"

	SDKActionRegistrationRequested EventType = "sdk:action_registration_requested"
	SDKActionRegistrationConfirmed EventType = "sdk:action_registration_confirmed"
	SDKCascadeTaskStarted          EventType = "sdk:cascade_task_started"
)

// EventDataKey identifies metadata entries.
type EventDataKey string

// EventData stores contextual attributes for an event.
type EventData map[EventDataKey]any

// Standard event data keys (mirroring SuperNode SDK).
const (
	KeyError            EventDataKey = "error"
	KeyCount            EventDataKey = "count"
	KeyTotal            EventDataKey = "total"
	KeySupernode        EventDataKey = "supernode"
	KeySupernodeAddress EventDataKey = "sn-address"
	KeyIteration        EventDataKey = "iteration"
	KeyTxHash           EventDataKey = "txhash"
	KeyMessage          EventDataKey = "message"
	KeyProgress         EventDataKey = "progress"
	KeyEventType        EventDataKey = "event_type"
	KeyOutputPath       EventDataKey = "output_path"

	KeyBytesTotal     EventDataKey = "bytes_total"
	KeyChunkSize      EventDataKey = "chunk_size"
	KeyEstChunks      EventDataKey = "est_chunks"
	KeyChunks         EventDataKey = "chunks"
	KeyElapsedSeconds EventDataKey = "elapsed_seconds"
	KeyThroughputMBS  EventDataKey = "throughput_mb_s"
	KeyChunkIndex     EventDataKey = "chunk_index"
	KeyReason         EventDataKey = "reason"
	KeyPrice          EventDataKey = "price"
	KeyExpiration     EventDataKey = "expiration_time"
	KeyFilePath       EventDataKey = "file_path"
	KeyBlockHeight    EventDataKey = "block_height"

	KeyTaskID   EventDataKey = "task_id"
	KeyActionID EventDataKey = "action_id"
)

// Event represents an emitted task event.
type Event struct {
	Type      EventType
	TaskID    string
	TaskType  string
	Timestamp time.Time
	ActionID  string
	Data      EventData
}

// Handler processes events. Context matches the subscription context used by the SDK.
type Handler func(ctx context.Context, e Event)
