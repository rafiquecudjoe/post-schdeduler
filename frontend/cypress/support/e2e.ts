// Cypress E2E support file

// Custom commands
declare global {
    namespace Cypress {
        interface Chainable {
            /**
             * Registers a new user
             */
            register(email: string, password: string): Chainable<void>;
            /**
             * Logs in a user
             */
            login(email: string, password: string): Chainable<void>;
            /**
             * Logs out the current user
             */
            logout(): Chainable<void>;
            /**
             * Creates a scheduled post
             */
            createPost(title: string, content: string, channel: string, scheduledAt: string): Chainable<void>;
        }
    }
}

// Register command
Cypress.Commands.add('register', (email: string, password: string) => {
    cy.visit('/register');
    cy.get('#email').type(email);
    cy.get('#password').type(password);
    cy.get('#confirmPassword').type(password);
    cy.get('button[type="submit"]').click();
    cy.url().should('include', '/dashboard');
});

// Login command
Cypress.Commands.add('login', (email: string, password: string) => {
    cy.visit('/login');
    cy.get('#email').type(email);
    cy.get('#password').type(password);
    cy.get('button[type="submit"]').click();
    cy.url().should('include', '/dashboard');
});

// Logout command
Cypress.Commands.add('logout', () => {
    cy.contains('button', 'Logout').click();
    cy.url().should('include', '/login');
});

// Create post command
Cypress.Commands.add('createPost', (title: string, content: string, channel: string, scheduledAt: string) => {
    cy.get('#title').clear().type(title);
    cy.get('#content').clear().type(content);
    cy.get('#channel').select(channel);
    cy.get('#scheduledAt').clear().type(scheduledAt);
    cy.contains('button', 'Schedule Post').click();
});

export { };
