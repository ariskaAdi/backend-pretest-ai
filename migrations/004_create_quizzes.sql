CREATE TYPE quiz_status AS ENUM ('pending', 'completed');

CREATE TABLE quizzes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id     UUID         NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    num_questions INT          NOT NULL,
    score         INT          DEFAULT NULL,
    status        quiz_status  NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE questions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id        UUID         NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    text           TEXT         NOT NULL,
    options        JSONB        NOT NULL,
    correct_answer VARCHAR(1)   NOT NULL,
    user_answer    VARCHAR(1)   DEFAULT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quizzes_user_id   ON quizzes(user_id);
CREATE INDEX idx_quizzes_module_id ON quizzes(module_id);
CREATE INDEX idx_questions_quiz_id ON questions(quiz_id);
