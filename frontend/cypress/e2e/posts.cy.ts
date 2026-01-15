describe('Post Scheduling', () => {
    const testEmail = `posts-${Date.now()}@example.com`;
    const testPassword = 'password123';

    before(() => {
        // Register a test user before all tests
        cy.register(testEmail, testPassword);
        cy.logout();
    });

    beforeEach(() => {
        // Login before each test
        cy.login(testEmail, testPassword);
    });

    describe('Create Post', () => {
        it('should show validation error for empty content', () => {
            cy.get('#title').type('Test Title');
            // Don't fill content
            const futureDate = new Date(Date.now() + 3600000).toISOString().slice(0, 16);
            cy.get('#scheduledAt').type(futureDate);
            cy.contains('button', 'Schedule Post').click();
            // HTML5 validation should prevent submission
            cy.get('#content:invalid').should('exist');
        });

        it('should create a scheduled post successfully', () => {
            const title = 'E2E Test Post';
            const content = 'This is a test post created by Cypress';
            const futureDate = new Date(Date.now() + 3600000).toISOString().slice(0, 16);

            cy.createPost(title, content, 'twitter', futureDate);

            // Form should be cleared
            cy.get('#title').should('have.value', '');
            cy.get('#content').should('have.value', '');

            // Post should appear in upcoming list
            cy.contains(title).should('be.visible');
            cy.contains(content).should('be.visible');
        });

        it('should create a post for each channel', () => {
            const channels = ['twitter', 'linkedin', 'facebook'];

            channels.forEach((channel) => {
                const title = `${channel.charAt(0).toUpperCase() + channel.slice(1)} Post`;
                const content = `Test post for ${channel}`;
                const futureDate = new Date(Date.now() + 3600000 + channels.indexOf(channel) * 60000).toISOString().slice(0, 16);

                cy.createPost(title, content, channel, futureDate);
                cy.contains(title).should('be.visible');
            });
        });
    });

    describe('Edit Post', () => {
        it('should edit a scheduled post', () => {
            // Create a post first
            const originalTitle = 'Original Title';
            const updatedTitle = 'Updated Title';
            const futureDate = new Date(Date.now() + 7200000).toISOString().slice(0, 16);

            cy.createPost(originalTitle, 'Original content', 'twitter', futureDate);
            cy.contains(originalTitle).should('be.visible');

            // Click edit button
            cy.contains(originalTitle).parents('[class*="bg-white"]').within(() => {
                cy.contains('Edit').click();
            });

            // Update the title in the edit form
            cy.get('input[type="text"]').first().clear().type(updatedTitle);
            cy.contains('Save').click();

            // Verify update
            cy.contains(updatedTitle).should('be.visible');
            cy.contains(originalTitle).should('not.exist');
        });
    });

    describe('Delete Post', () => {
        it('should delete a scheduled post', () => {
            // Create a post first
            const titleToDelete = 'Post to Delete';
            const futureDate = new Date(Date.now() + 7200000).toISOString().slice(0, 16);

            cy.createPost(titleToDelete, 'This will be deleted', 'linkedin', futureDate);
            cy.contains(titleToDelete).should('be.visible');

            // Click delete button
            cy.contains(titleToDelete).parents('[class*="bg-white"]').within(() => {
                cy.contains('Delete').click();
            });

            // Confirm deletion (if there's a confirmation dialog)
            // cy.contains('Confirm').click();

            // Verify post is gone
            cy.contains(titleToDelete).should('not.exist');
        });
    });

    describe('View Tabs', () => {
        it('should switch between Upcoming and History tabs', () => {
            // Check Upcoming tab is active by default
            cy.contains('button', 'Upcoming').should('have.class', 'text-blue-600');

            // Switch to History tab
            cy.contains('button', 'History').click();
            cy.contains('button', 'History').should('have.class', 'text-blue-600');

            // Switch back to Upcoming
            cy.contains('button', 'Upcoming').click();
            cy.contains('button', 'Upcoming').should('have.class', 'text-blue-600');
        });
    });
});
