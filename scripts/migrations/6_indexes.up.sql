create index idx_simple_transactions_pulse_record
    on simple_transactions (pulse_record);

create index idx_simple_transactions_finish_pulse_record
    on simple_transactions (finish_pulse_record);

create index idx_members_account_state
    on members (account_state);
