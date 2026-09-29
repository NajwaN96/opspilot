DROP INDEX IF EXISTS rollouts_one_payment_canary;

CREATE UNIQUE INDEX rollouts_one_payment_canary
    ON rollouts (namespace, service)
    WHERE state IN (
        'PENDING',
        'RUNNING',
        'ANALYZING',
        'AWAITING_APPROVAL',
        'PROMOTING',
        'ROLLING_BACK',
        'NEEDS_ATTENTION'
    );
