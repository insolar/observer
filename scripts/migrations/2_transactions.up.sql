drop type if exists transaction_type;
create type transaction_type as enum ('transfer', 'migration', 'release');

create table simple_transactions
(
    id bigserial not null constraint simple_transactions_pkey primary key,
    tx_id bytea not null constraint simple_transactions_tx_id_key unique,

    status_registered bool,
    type transaction_type,
    pulse_record bigint[2] unique,
    member_from_ref bytea,
    member_to_ref bytea,
    deposit_to_ref bytea,
    deposit_from_ref bytea,
    amount varchar(256),

    status_sent bool,

    status_finished bool,
    finish_success bool,
    finish_pulse_record bigint[2] unique,
    fee varchar(256)
);
