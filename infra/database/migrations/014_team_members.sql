-- =====================================================
-- MODULE 15: Team Members
-- =====================================================
-- team_members (core)
CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(255),
    job_title VARCHAR(255),
    department VARCHAR(100),
    team VARCHAR(100),
    work_email VARCHAR(255),
    work_phone VARCHAR(20),
    office_location VARCHAR(255),
    profile_picture_file_id UUID REFERENCES files(id),
    timezone VARCHAR(50),
    work_hours_start TIME DEFAULT '09:00',
    work_hours_end TIME DEFAULT '17:00',
    work_days INTEGER[] DEFAULT ARRAY[1,2,3,4,5],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- teams
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    team_lead_id UUID REFERENCES team_members(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- team_memberships
CREATE TABLE team_memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    membership_type VARCHAR(20) DEFAULT 'member',
    is_primary_team BOOLEAN DEFAULT false,
    joined_team_at DATE DEFAULT CURRENT_DATE,
    left_team_at DATE,
    is_active BOOLEAN DEFAULT true,
    UNIQUE(team_id, team_member_id)
);

-- team_member_availability
CREATE TABLE team_member_availability (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'available',
    start_time TIME,
    end_time TIME,
    is_all_day BOOLEAN DEFAULT false,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

