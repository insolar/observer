create table simple_transactions
(
    id bigserial not null constraint simple_transactions_pkey primary key,
    tx_id bytea not null constraint simple_transactions_tx_id_key unique,

    status_registered bool,
    pulse_number bigint,
    record_number bigint,
    member_from_ref bytea,
    member_to_ref bytea,
    migration_to_ref bytea,
    vesting_from_ref bytea,
    amount varchar(256),
    fee varchar(256),

    status_sent bool,

    status_finished bool,
    finish_success bool,
    finish_pulse_number bigint,
    finish_record_number bigint
);
