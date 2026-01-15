describe('Authentication', () => {
    const testEmail = `test-${Date.now()}@example.com`;
    const testPassword = 'password123';

    beforeEach(() => {
        // Clear cookies before each test
        cy.clearCookies();
    });

    describe('Registration', () => {
        it('should show validation errors for empty form', () => {
            cy.visit('/register');
            cy.get('button[type="submit"]').click();
            // HTML5 validation should prevent submission
            cy.url().should('include', '/register');
        });

        it('should show error for password mismatch', () => {
            cy.visit('/register');
            cy.get('#email').type(testEmail);
            cy.get('#password').type(testPassword);
            cy.get('#confirmPassword').type('different123');
            cy.get('button[type="submit"]').click();
            cy.contains('Passwords do not match').should('be.visible');
        });

        it('should register a new user successfully', () => {
            cy.register(testEmail, testPassword);
            cy.url().should('include', '/dashboard');
            cy.contains(testEmail).should('be.visible');
        });

        it('should show error for duplicate email', () => {
            cy.visit('/register');
            cy.get('#email').type(testEmail);
            cy.get('#password').type(testPassword);
            cy.get('#confirmPassword').type(testPassword);
            cy.get('button[type="submit"]').click();
            cy.contains('already exists').should('be.visible');
        });
    });

    describe('Login', () => {
        it('should show validation errors for empty form', () => {
            cy.visit('/login');
            cy.get('button[type="submit"]').click();
            cy.url().should('include', '/login');
        });

        it('should show error for invalid credentials', () => {
            cy.visit('/login');
            cy.get('#email').type('invalid@example.com');
            cy.get('#password').type('wrongpassword');
            cy.get('button[type="submit"]').click();
            cy.contains('Invalid').should('be.visible');
        });

        it('should login successfully with valid credentials', () => {
            cy.login(testEmail, testPassword);
            cy.url().should('include', '/dashboard');
            cy.contains(testEmail).should('be.visible');
        });
    });

    describe('Logout', () => {
        it('should logout and redirect to login', () => {
            cy.login(testEmail, testPassword);
            cy.logout();
            cy.url().should('include', '/login');
        });
    });

    describe('Protected Routes', () => {
        it('should redirect unauthenticated users to login', () => {
            cy.visit('/dashboard');
            cy.url().should('include', '/login');
        });

        it('should redirect authenticated users away from login', () => {
            cy.login(testEmail, testPassword);
            cy.visit('/login');
            cy.url().should('include', '/dashboard');
        });
    });
});
