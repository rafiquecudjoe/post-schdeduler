describe('Full Scheduling Flow', () => {
    const testEmail = `flow-${Date.now()}@example.com`;
    const testPassword = 'password123';

    before(() => {
        cy.register(testEmail, testPassword);
    });

    it('should complete the full scheduling → publishing flow', () => {
        // Create a post scheduled for 1 minute from now
        const postTitle = 'Quick Publish Test';
        const postContent = 'This post should be published by the worker';

        // Schedule for 1 minute from now
        const scheduledTime = new Date(Date.now() + 60000);
        const scheduledAt = scheduledTime.toISOString().slice(0, 16);

        cy.createPost(postTitle, postContent, 'twitter', scheduledAt);

        // Verify post appears in Upcoming
        cy.contains('button', 'Upcoming').click();
        cy.contains(postTitle).should('be.visible');
        cy.contains('scheduled').should('be.visible');

        // Wait for the worker to publish (wait slightly longer than 1 minute + worker poll interval)
        // Note: In real tests, you might use cy.clock() to speed this up
        cy.log('Waiting for worker to publish the post (this may take ~70 seconds)...');
        cy.wait(75000); // 75 seconds

        // Refresh the page to get updated data
        cy.reload();

        // Check History tab for published post
        cy.contains('button', 'History').click();
        cy.contains(postTitle).should('be.visible');
        cy.contains('published').should('be.visible');

        // Verify it's no longer in Upcoming
        cy.contains('button', 'Upcoming').click();
        cy.contains(postTitle).should('not.exist');
    });
});
