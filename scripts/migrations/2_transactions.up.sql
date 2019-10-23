create type transaction_type as enum ('transfer', 'migration', 'vesting');
create type transaction_status as enum ('registered', 'sent', 'received');

create table simple_transactions
(
    id bigserial not null constraint simple_transactions_pkey primary key,
    tx_id bytea constraint simple_transactions_tx_id_key unique,
    pulse_number bigint not null,

    type transaction_type not null,
    status transaction_status not null,

    member_from_ref bytea,
    member_to_ref bytea,
    migration_to_ref bytea,
    vesting_from_ref bytea,

    amount varchar(256) not null,
    fee varchar(256) not null
);
